package runner

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/aschepis/maslow/internal/spec"
)

// maxMultipartBody caps a multipart body, which is held in memory so a
// redirect can replay it.
const maxMultipartBody = 10 << 20

const defaultPartContentType = "application/octet-stream"

// fixtureError is a problem with a step's input files. Retrying cannot fix
// it, so the scenario's status is "error" rather than "fail".
type fixtureError struct{ err error }

func (e *fixtureError) Error() string { return e.err.Error() }
func (e *fixtureError) Unwrap() error { return e.err }

// resolvedPart is a part with its bytes, ready to encode.
type resolvedPart struct {
	field       string
	isFile      bool
	filename    string
	contentType string
	data        []byte
}

// checkMultipartStep rejects steps that would send a body Maslow did not build.
func checkMultipartStep(step spec.Step) error {
	if step.Body != "" {
		return errors.New("body and multipart are both set; use one")
	}
	for name := range step.Headers {
		if strings.EqualFold(name, "Content-Type") {
			return fmt.Errorf("header %q conflicts with multipart, which sets Content-Type itself", name)
		}
	}
	return nil
}

// buildMultipart encodes parts as a multipart/form-data body and returns it
// with its Content-Type. Files are read from the directory Maslow runs in.
func buildMultipart(parts []spec.Part) (body []byte, contentType string, err error) {
	root, err := os.OpenRoot(".")
	if err != nil {
		return nil, "", &fixtureError{err}
	}
	defer root.Close()

	resolved := make([]resolvedPart, len(parts))
	size := 0
	for i, p := range parts {
		r, err := resolvePart(p, root, maxMultipartBody-size)
		if err != nil {
			return nil, "", fmt.Errorf("multipart[%d] (field %q): %w", i, p.Field, err)
		}
		resolved[i] = r
		size += len(r.data)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.SetBoundary(boundaryFor(resolved)); err != nil {
		return nil, "", err
	}
	for _, r := range resolved {
		if err := writePart(w, r); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	if buf.Len() > maxMultipartBody {
		return nil, "", &fixtureError{errors.New("multipart body exceeds the 10 MiB cap")}
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}

func resolvePart(p spec.Part, root *os.Root, room int) (resolvedPart, error) {
	r := resolvedPart{field: p.Field, filename: p.Filename, contentType: p.ContentType}
	switch {
	case p.File != "":
		data, err := readFixture(root, p.File, room)
		if err != nil {
			return r, &fixtureError{fmt.Errorf("file %q: %w", p.File, err)}
		}
		r.isFile, r.data = true, data
		if r.filename == "" {
			r.filename = path.Base(p.File)
		}
	case p.Content != "" || p.Filename != "":
		r.isFile, r.data = true, []byte(p.Content)
	default:
		r.data = []byte(p.Value)
	}
	if r.isFile && r.contentType == "" {
		r.contentType = defaultPartContentType
	}
	if len(r.data) > room {
		return r, &fixtureError{errors.New("multipart body exceeds the 10 MiB cap")}
	}
	return r, nil
}

// readFixture reads at most room+1 bytes, enough to tell the file is too big.
func readFixture(root *os.Root, name string, room int) ([]byte, error) {
	if err := checkFixturePath(name); err != nil {
		return nil, err
	}
	f, err := root.Open(filepath.FromSlash(name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, int64(room)+1))
}

// checkFixturePath enforces the schema's rule for runs that bypass it.
func checkFixturePath(name string) error {
	if strings.Contains(name, `\`) {
		return errors.New("use forward slashes")
	}
	for _, seg := range strings.Split(name, "/") {
		switch seg {
		case "":
			return errors.New("must be relative, with no empty segment")
		case ".", "..":
			return fmt.Errorf("must not contain %q", seg)
		}
	}
	return nil
}

// boundaryFor derives the boundary from the parts, so the same parts always
// produce the same request bytes.
func boundaryFor(parts []resolvedPart) string {
	h := sha256.New()
	for _, p := range parts {
		for _, s := range []string{p.field, p.filename, p.contentType, string(p.data)} {
			_ = binary.Write(h, binary.BigEndian, uint64(len(s)))
			h.Write([]byte(s))
		}
	}
	return "maslow-" + hex.EncodeToString(h.Sum(nil))[:32]
}

var quoteEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func writePart(w *multipart.Writer, p resolvedPart) error {
	if !p.isFile {
		return w.WriteField(p.field, string(p.data))
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
		quoteEscaper.Replace(p.field), quoteEscaper.Replace(p.filename)))
	header.Set("Content-Type", p.contentType)
	part, err := w.CreatePart(header)
	if err != nil {
		return err
	}
	_, err = part.Write(p.data)
	return err
}
