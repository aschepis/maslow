package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aschepis/maslow/internal/evidence"
	"github.com/aschepis/maslow/internal/spec"
)

type receivedPart struct {
	Field       string `json:"field"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Value       string `json:"value"`
	Len         int    `json:"len"`
	SHA256      string `json:"sha256"`
}

type uploadReply struct {
	ContentType string         `json:"content_type"`
	Parts       []receivedPart `json:"parts"`
}

// uploadServer echoes the parts it receives at /upload and counts requests
// there, so a test can tell whether anything was sent.
func uploadServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var uploads atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		uploads.Add(1)
		writeUploadReply(t, w, r)
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
		w.Write([]byte(`{"id":"42"}`))
	})
	mux.HandleFunc("/moved", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/private-upload", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/private-upload", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err != nil || c.Value != "abc" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		writeUploadReply(t, w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, &uploads
}

func writeUploadReply(t *testing.T, w http.ResponseWriter, r *http.Request) {
	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reply := uploadReply{ContentType: r.Header.Get("Content-Type"), Parts: []receivedPart{}}
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, _ := io.ReadAll(p)
		sum := sha256.Sum256(data)
		got := receivedPart{
			Field:       p.FormName(),
			Filename:    p.FileName(),
			ContentType: p.Header.Get("Content-Type"),
			Len:         len(data),
			SHA256:      hex.EncodeToString(sum[:]),
		}
		if got.Filename == "" {
			got.Value = string(data)
		}
		reply.Parts = append(reply.Parts, got)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reply)
}

// inFixtureDir runs the test in a fresh directory holding files, the way
// Maslow runs in a checkout.
func inFixtureDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(dir)
	return dir
}

func uploadStep(url string, parts ...spec.Part) spec.Step {
	return spec.Step{Action: "http", Method: "POST", URL: url, Multipart: parts}
}

func runUpload(steps ...spec.Step) evidence.ContractResult {
	return RunContracts([]spec.Contract{{Name: "c", Scenarios: []spec.Scenario{{Name: "s", Steps: steps}}}})[0]
}

// lastReply decodes the reply a scenario's final HTTP step received, by
// failing that step on purpose so the evidence carries its output.
func lastReply(t *testing.T, got evidence.ContractResult) uploadReply {
	t.Helper()
	var reply uploadReply
	if err := json.Unmarshal([]byte(got.Output), &reply); err != nil {
		t.Fatalf("expected the server's JSON reply in output, got %s: %s (%v)", got.Status, got.Output, err)
	}
	return reply
}

// failOn makes a step fail after the request is sent, so its reply lands in evidence.
func failOn(step spec.Step) spec.Step {
	never := 599
	step.Expect = &spec.Expectation{Status: &never}
	return step
}

func sha(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestMultipart_SingleFile(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, map[string]string{"fixtures/clip.bin": "binary-ish"})

	got := runUpload(failOn(uploadStep(srv.URL+"/upload", spec.Part{Field: "video", File: "fixtures/clip.bin"})))

	reply := lastReply(t, got)
	want := []receivedPart{{Field: "video", Filename: "clip.bin", ContentType: "application/octet-stream", Len: 10, SHA256: sha("binary-ish")}}
	if !slices.Equal(reply.Parts, want) {
		t.Errorf("expected %+v, got %+v", want, reply.Parts)
	}
	if !strings.HasPrefix(reply.ContentType, "multipart/form-data; boundary=maslow-") {
		t.Errorf("expected a maslow multipart boundary, got %q", reply.ContentType)
	}
}

func TestMultipart_MixedPartsArriveInSpecOrder(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, map[string]string{"a.txt": "first file", "dir/b.txt": "second"})

	got := runUpload(failOn(uploadStep(srv.URL+"/upload",
		spec.Part{Field: "title", Value: "hello"},
		spec.Part{Field: "one", File: "a.txt"},
		spec.Part{Field: "kind", Value: "clip"},
		spec.Part{Field: "two", File: "dir/b.txt", ContentType: "text/plain"},
	)))

	want := []receivedPart{
		{Field: "title", Value: "hello", Len: 5, SHA256: sha("hello")},
		{Field: "one", Filename: "a.txt", ContentType: "application/octet-stream", Len: 10, SHA256: sha("first file")},
		{Field: "kind", Value: "clip", Len: 4, SHA256: sha("clip")},
		{Field: "two", Filename: "b.txt", ContentType: "text/plain", Len: 6, SHA256: sha("second")},
	}
	if parts := lastReply(t, got).Parts; !slices.Equal(parts, want) {
		t.Errorf("expected %+v, got %+v", want, parts)
	}
}

func TestMultipart_InlineContent(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, nil)

	got := runUpload(failOn(uploadStep(srv.URL+"/upload",
		spec.Part{Field: "notes", Content: "inline text", Filename: "notes.md", ContentType: "text/markdown"},
	)))

	want := []receivedPart{{Field: "notes", Filename: "notes.md", ContentType: "text/markdown", Len: 11, SHA256: sha("inline text")}}
	if parts := lastReply(t, got).Parts; !slices.Equal(parts, want) {
		t.Errorf("expected %+v, got %+v", want, parts)
	}
}

func TestMultipart_TemplatesValuesAndRedactsEnv(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, nil)
	t.Setenv("MASLOW_TEST_TOKEN", "s3cret-token")

	got := RunContracts([]spec.Contract{{Name: "c", Env: []string{"MASLOW_TEST_TOKEN"}, Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{
		{Action: "http", Method: "POST", URL: srv.URL + "/login"},
		{Action: "capture", From: "$.id", As: "id"},
		failOn(uploadStep(srv.URL+"/upload",
			spec.Part{Field: "id", Value: "{{.cap.id}}"},
			spec.Part{Field: "token", Value: "{{.env.MASLOW_TEST_TOKEN}}"},
			spec.Part{Field: "doc", Content: "for {{.cap.id}}", Filename: "{{.cap.id}}.txt"},
		)),
	}}}}})[0]

	if strings.Contains(got.Output, "s3cret-token") || !strings.Contains(got.Output, "[redacted:MASLOW_TEST_TOKEN]") {
		t.Errorf("expected the echoed token to be redacted, got %s", got.Output)
	}
	parts := lastReply(t, got).Parts
	if len(parts) != 3 || parts[0].Value != "42" || parts[2].Filename != "42.txt" || parts[2].SHA256 != sha("for 42") {
		t.Errorf("expected captured values substituted, got %+v", parts)
	}
}

func TestMultipart_MissingFileIsAnErrorAndNotRetried(t *testing.T) {
	srv, uploads := uploadServer(t)
	dir := inFixtureDir(t, nil)

	got := RunContracts([]spec.Contract{{Name: "c", Retries: 2, Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{
		uploadStep(srv.URL+"/upload", spec.Part{Field: "title", Value: "x"}, spec.Part{Field: "video", File: "fixtures/missing.bin"}),
	}}}}})[0]

	if got.Status != "error" {
		t.Fatalf("expected status error, got %s: %s", got.Status, got.Error)
	}
	if got.Attempts != nil {
		t.Errorf("expected one attempt, got %+v", got.Attempts)
	}
	want := evidence.FailedStep{Phase: "steps", Index: 0, Action: "http", Target: "POST " + srv.URL + "/upload"}
	if got.FailedStep == nil || got.FailedStep.Phase != want.Phase || got.FailedStep.Index != want.Index ||
		got.FailedStep.Action != want.Action || got.FailedStep.Target != want.Target {
		t.Errorf("expected failed_step %+v, got %+v", want, got.FailedStep)
	}
	for _, s := range []string{`multipart[1] (field "video")`, `"fixtures/missing.bin"`} {
		if !strings.Contains(got.Error, s) {
			t.Errorf("expected error to contain %s, got %s", s, got.Error)
		}
	}
	if strings.Contains(got.Error, dir) {
		t.Errorf("expected no absolute path in the error, got %s", got.Error)
	}
	if n := uploads.Load(); n != 0 {
		t.Errorf("expected nothing sent, got %d requests", n)
	}
}

func TestMultipart_PathsOutsideTheCheckoutAreErrors(t *testing.T) {
	srv, uploads := uploadServer(t)
	outside := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(outside, []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := inFixtureDir(t, nil)
	if err := os.Symlink(outside, filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}

	for _, file := range []string{"../secret", outside, "a/../link", "link"} {
		t.Run(file, func(t *testing.T) {
			got := runUpload(uploadStep(srv.URL+"/upload", spec.Part{Field: "f", File: file}))
			if got.Status != "error" || !strings.Contains(got.Error, `multipart[0] (field "f")`) {
				t.Errorf("expected an error naming the part, got %s: %s", got.Status, got.Error)
			}
		})
	}
	if n := uploads.Load(); n != 0 {
		t.Errorf("expected nothing sent, got %d requests", n)
	}
}

func TestMultipart_ContentTypeHeaderFailsBeforeSending(t *testing.T) {
	srv, uploads := uploadServer(t)
	inFixtureDir(t, nil)

	step := uploadStep(srv.URL+"/upload", spec.Part{Field: "title", Value: "x"})
	step.Headers = map[string]string{"content-type": "text/plain"}
	got := runUpload(step)

	if got.Status != "fail" || !strings.Contains(got.Error, `"content-type"`) {
		t.Errorf("expected a failure naming the header, got %s: %s", got.Status, got.Error)
	}
	if n := uploads.Load(); n != 0 {
		t.Errorf("expected nothing sent, got %d requests", n)
	}
}

func TestMultipart_BodyAndMultipartFailBeforeSending(t *testing.T) {
	srv, uploads := uploadServer(t)
	inFixtureDir(t, nil)

	step := uploadStep(srv.URL+"/upload", spec.Part{Field: "title", Value: "x"})
	step.Body = "raw"
	got := runUpload(step)

	if got.Status != "fail" || !strings.Contains(got.Error, "body and multipart") {
		t.Errorf("expected a failure naming body and multipart, got %s: %s", got.Status, got.Error)
	}
	if n := uploads.Load(); n != 0 {
		t.Errorf("expected nothing sent, got %d requests", n)
	}
}

func TestMultipart_BodyOverTheCapIsAnError(t *testing.T) {
	srv, uploads := uploadServer(t)
	dir := inFixtureDir(t, map[string]string{"big.bin": ""})
	if err := os.Truncate(filepath.Join(dir, "big.bin"), 10<<20+1); err != nil {
		t.Fatal(err)
	}

	got := runUpload(uploadStep(srv.URL+"/upload", spec.Part{Field: "f", File: "big.bin"}))

	if got.Status != "error" || !strings.Contains(got.Error, "10 MiB") {
		t.Errorf("expected an error naming the cap, got %s: %s", got.Status, got.Error)
	}
	if n := uploads.Load(); n != 0 {
		t.Errorf("expected nothing sent, got %d requests", n)
	}
}

func TestMultipart_IsDeterministic(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, map[string]string{"a.txt": "same bytes"})
	step := failOn(uploadStep(srv.URL+"/upload", spec.Part{Field: "t", Value: "v"}, spec.Part{Field: "f", File: "a.txt"}))

	first, second := runUpload(step), runUpload(step)

	if a, b := lastReply(t, first).ContentType, lastReply(t, second).ContentType; a != b {
		t.Errorf("expected the same Content-Type twice, got %q and %q", a, b)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Errorf("expected identical results, got\n%s\n%s", a, b)
	}
}

// Fixtures are test input, so a red-green base run still reads them from the head checkout.
func TestMultipart_ReadsFixturesFromTheWorkingDirectoryNotDir(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, map[string]string{"a.txt": "head"})
	ok := 200

	step := uploadStep(srv.URL+"/upload", spec.Part{Field: "f", File: "a.txt"})
	step.Expect = &spec.Expectation{Status: &ok, BodyContains: sha("head")}
	got := RunContractsWith([]spec.Contract{{Name: "c", Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{step}}}}},
		ContractOptions{Dir: t.TempDir()})[0]

	if got.Status != "pass" {
		t.Errorf("expected the fixture read from the working directory, got %s: %s", got.Status, got.Error)
	}
}

func TestMultipart_RedirectReplaysBodyWithSessionCookie(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, map[string]string{"a.txt": "replayed"})
	ok := 200

	step := uploadStep(srv.URL+"/moved", spec.Part{Field: "f", File: "a.txt"})
	step.Expect = &spec.Expectation{Status: &ok, BodyContains: sha("replayed")}
	got := runUpload(spec.Step{Action: "http", Method: "POST", URL: srv.URL + "/login"}, step)

	if got.Status != "pass" {
		t.Errorf("expected the 307 to replay the multipart body with the cookie, got %s: %s %s", got.Status, got.Error, got.Output)
	}
}

func TestMultipart_RelativeURLUsesBaseURL(t *testing.T) {
	srv, _ := uploadServer(t)
	inFixtureDir(t, nil)
	ok := 200

	step := uploadStep("/upload", spec.Part{Field: "t", Value: "v"})
	step.Expect = &spec.Expectation{Status: &ok, BodyContains: `"value":"v"`}
	got := RunContracts([]spec.Contract{{Name: "c", BaseURL: srv.URL, Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{step}}}}})[0]

	if got.Status != "pass" {
		t.Errorf("expected the relative URL joined to base_url, got %s: %s", got.Status, got.Error)
	}
}
