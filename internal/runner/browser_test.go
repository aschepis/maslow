package runner

import (
	"net/http"
	"net/http/httptest"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/aschepis/maslow/internal/spec"
)

// requireBrowser skips tests that drive a real browser on machines without one.
func requireBrowser(t *testing.T) {
	t.Helper()
	if _, err := findBrowser(); err != nil {
		t.Skipf("no browser for browser steps: %v", err)
	}
}

const greetPage = `<!doctype html>
<title>greet</title>
<input id="name">
<button id="go" onclick="location.href='/hello?name='+encodeURIComponent(document.getElementById('name').value)">Go</button>
<button id="off" disabled>Off</button>
<div id="late"></div>
<script>
console.error("boom from the page");
setTimeout(function () { document.getElementById("late").innerHTML = '<button id="later">Later</button>'; }, 300);
</script>`

// uiServer serves a small form, a greeting, and a page that needs the session cookie.
func uiServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(greetPage))
	})
	mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p id="greeting">Hello, ` + r.URL.Query().Get("name") + `</p>`))
	})
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: "session", Value: "abc", Path: "/"})
	})
	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("session"); err != nil || c.Value != "abc" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "theme", Value: "dark", Path: "/"})
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<p id="who">signed in</p>`))
	})
	mux.HandleFunc("/theme", func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("theme"); err != nil || c.Value != "dark" {
			w.WriteHeader(http.StatusUnauthorized)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func browserStep(do string, f func(*spec.Step)) spec.Step {
	s := spec.Step{Action: "browser", Do: do}
	if f != nil {
		f(&s)
	}
	return s
}

func yes() *bool { b := true; return &b }
func no() *bool  { b := false; return &b }

func TestBrowser_DrivesAPageThroughABaseURL(t *testing.T) {
	requireBrowser(t)
	srv := uiServer(t)
	contract := spec.Contract{Name: "ui", BaseURL: "http://unused.invalid", Scenarios: []spec.Scenario{{Name: "greet", Steps: []spec.Step{
		browserStep("navigate", func(s *spec.Step) { s.URL = "/" }),
		browserStep("fill", func(s *spec.Step) { s.Selector = "#name"; s.Text = "Ada" }),
		browserStep("click", func(s *spec.Step) { s.Selector = "#go" }),
		browserStep("wait", func(s *spec.Step) { s.Selector = "#greeting"; s.Timeout = "5s" }),
		browserStep("assert", func(s *spec.Step) {
			s.Selector = "#greeting"
			s.Expect = &spec.Expectation{TextContains: "Hello, Ada", URLContains: "/hello?name=Ada", Visible: yes(), Enabled: yes()}
		}),
	}}}}

	got := RunContractsWith([]spec.Contract{contract}, ContractOptions{BaseURL: srv.URL})[0]
	if got.Status != "pass" {
		t.Fatalf("expected pass, got %s: %s (failed step %+v)", got.Status, got.Error, got.FailedStep)
	}
}

func TestBrowser_FailingStepRecordsPageConsoleAndScreenshot(t *testing.T) {
	requireBrowser(t)
	srv := uiServer(t)
	t.Chdir(t.TempDir())
	contract := spec.Contract{Name: "ui", BaseURL: srv.URL, Scenarios: []spec.Scenario{{Name: "broken page", Steps: []spec.Step{
		browserStep("navigate", func(s *spec.Step) { s.URL = "/" }),
		browserStep("assert", func(s *spec.Step) {
			s.Selector = "#off"
			s.Expect = &spec.Expectation{Enabled: yes()}
		}),
	}}}}

	got := RunContracts([]spec.Contract{contract})[0]
	if got.Status != "fail" {
		t.Fatalf("expected fail, got %s: %s", got.Status, got.Error)
	}
	step := got.FailedStep
	if step == nil || step.Action != "browser" || step.Index != 1 || step.Target != "assert #off" {
		t.Fatalf("failed_step = %+v, want browser assert #off at index 1", step)
	}
	if step.PageURL != srv.URL+"/" {
		t.Errorf("page_url = %q, want %q", step.PageURL, srv.URL+"/")
	}
	if !slices.ContainsFunc(step.Console, func(m string) bool { return strings.Contains(m, "boom from the page") }) {
		t.Errorf("console = %q, want the page's console.error", step.Console)
	}
	const shot = "reports/browser/ui/broken-page.png"
	if !slices.Contains(got.Artifacts, shot) {
		t.Errorf("artifacts = %q, want %s", got.Artifacts, shot)
	}
	if info, err := os.Stat(shot); err != nil || info.Size() == 0 {
		t.Errorf("screenshot %s not written: %v", shot, err)
	}
}

func TestBrowser_PassingRunRemovesAStaleScreenshot(t *testing.T) {
	requireBrowser(t)
	srv := uiServer(t)
	t.Chdir(t.TempDir())
	const shot = "reports/browser/ui/s.png"
	if err := os.MkdirAll("reports/browser/ui", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(shot, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	contract := spec.Contract{Name: "ui", BaseURL: srv.URL, Scenarios: []spec.Scenario{{
		Name:      "s",
		Artifacts: []string{"reports/browser/**"},
		Steps:     []spec.Step{browserStep("navigate", func(s *spec.Step) { s.URL = "/" })},
	}}}

	got := RunContracts([]spec.Contract{contract})[0]
	if got.Status != "pass" || len(got.Artifacts) != 0 {
		t.Errorf("got %s with artifacts %q, want pass and no stale screenshot", got.Status, got.Artifacts)
	}
}

func TestBrowser_ActionsDoNotWaitImplicitly(t *testing.T) {
	requireBrowser(t)
	srv := uiServer(t)
	run := func(steps ...spec.Step) string {
		t.Chdir(t.TempDir())
		c := spec.Contract{Name: "ui", BaseURL: srv.URL, Scenarios: []spec.Scenario{{Name: "s", Steps: steps}}}
		got := RunContracts([]spec.Contract{c})[0]
		return got.Status + ": " + got.Error
	}
	nav := browserStep("navigate", func(s *spec.Step) { s.URL = "/" })
	click := browserStep("click", func(s *spec.Step) { s.Selector = "#later" })
	wait := browserStep("wait", func(s *spec.Step) { s.Selector = "#later"; s.Timeout = "5s" })

	if got := run(nav, click); !strings.HasPrefix(got, "fail") || !strings.Contains(got, `"#later" not found`) {
		t.Errorf("click before the element exists: got %q, want an immediate not-found failure", got)
	}
	if got := run(nav, wait, click); !strings.HasPrefix(got, "pass") {
		t.Errorf("click after an explicit wait: got %q, want pass", got)
	}
}

func TestBrowser_SharesTheScenarioSessionWithHTTPSteps(t *testing.T) {
	requireBrowser(t)
	srv := uiServer(t)
	ok := 200
	contract := spec.Contract{Name: "ui", BaseURL: srv.URL, Scenarios: []spec.Scenario{{
		Name:  "session",
		Setup: []spec.Step{{Action: "http", Method: "POST", URL: "/login", Expect: &spec.Expectation{Status: &ok}}},
		Steps: []spec.Step{
			browserStep("navigate", func(s *spec.Step) { s.URL = "/me" }),
			browserStep("assert", func(s *spec.Step) { s.Expect = &spec.Expectation{TextContains: "signed in"} }),
			{Action: "http", Method: "GET", URL: "/theme", Expect: &spec.Expectation{Status: &ok}},
		},
	}}}

	if got := RunContracts([]spec.Contract{contract})[0]; got.Status != "pass" {
		t.Errorf("expected cookies to flow between HTTP and browser steps, got %s: %s", got.Status, got.Error)
	}
}

// These need no browser: they fail before one would start.
func TestBrowser_FailsClearlyWithoutABrowser(t *testing.T) {
	t.Setenv("MASLOW_BROWSER", "/nonexistent/chrome")
	contract := spec.Contract{Name: "ui", Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{
		browserStep("navigate", func(s *spec.Step) { s.URL = "http://127.0.0.1:1/" }),
	}}}}

	got := RunContracts([]spec.Contract{contract})[0]
	if got.Status != "fail" || !strings.Contains(got.Error, "MASLOW_BROWSER") {
		t.Errorf("got %s: %q, want a failure naming MASLOW_BROWSER", got.Status, got.Error)
	}
}

func TestBrowser_RejectsStateAssertionsWithoutASelector(t *testing.T) {
	t.Setenv("MASLOW_BROWSER", "/nonexistent/chrome")
	for _, expect := range []*spec.Expectation{{Visible: no()}, {Enabled: yes()}, {}} {
		contract := spec.Contract{Name: "ui", Scenarios: []spec.Scenario{{Name: "s", Steps: []spec.Step{
			browserStep("assert", func(s *spec.Step) { s.Expect = expect }),
		}}}}
		got := RunContracts([]spec.Contract{contract})[0]
		if got.Status != "fail" || strings.Contains(got.Error, "MASLOW_BROWSER") {
			t.Errorf("expect %+v: got %s: %q, want a spec error before the browser starts", expect, got.Status, got.Error)
		}
	}
}
