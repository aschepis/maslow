package runner

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/aschepis/maslow-agentic/internal/spec"
)

func TestRunRefs_Empty(t *testing.T) {
	results := RunRefs(nil)
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestRunRefs_FileRefExists(t *testing.T) {
	// Create a temp file.
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	os.WriteFile(path, []byte("hello"), 0o644)

	refs := []spec.Ref{
		{Kind: "file", Path: path, Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestRunRefs_FileRefMissingRequired(t *testing.T) {
	refs := []spec.Ref{
		{Kind: "file", Path: "/nonexistent/path/file.txt", Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestRunRefs_FileRefMissingOptional(t *testing.T) {
	refs := []spec.Ref{
		{Kind: "file", Path: "/nonexistent/path/file.txt", Required: false},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skip" {
		t.Errorf("expected skip, got %s", results[0].Status)
	}
}

func TestRunRefs_BinaryRefExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mybin")
	os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755)

	refs := []spec.Ref{
		{Kind: "binary", Path: path, Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestRunRefs_BinaryRefNotExecutable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mybin")
	os.WriteFile(path, []byte("not executable"), 0o644)

	refs := []spec.Ref{
		{Kind: "binary", Path: path, Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestRunRefs_BinaryRefMissing(t *testing.T) {
	refs := []spec.Ref{
		{Kind: "binary", Path: "/nonexistent/bin", Required: false},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skip" {
		t.Errorf("expected skip, got %s", results[0].Status)
	}
}

func TestRunRefs_URLRefPass(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	refs := []spec.Ref{
		{Kind: "url", Path: server.URL, Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}

func TestRunRefs_URLRefFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	refs := []spec.Ref{
		{Kind: "url", Path: server.URL, Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "fail" {
		t.Errorf("expected fail, got %s", results[0].Status)
	}
}

func TestRunRefs_URLRefOptionalFail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	refs := []spec.Ref{
		{Kind: "url", Path: server.URL, Required: false},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "skip" {
		t.Errorf("expected skip, got %s", results[0].Status)
	}
}

func TestRunRefs_UnknownKind(t *testing.T) {
	refs := []spec.Ref{
		{Kind: "package", Path: "github.com/foo/bar"},
		{Kind: "container", Path: "myimage:latest"},
		{Kind: "mcp", Path: "@anthropic/mcp-browser", Name: "browser"},
	}
	results := RunRefs(refs)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Status != "skip" {
			t.Errorf("expected skip for kind %s, got %s", r.Kind, r.Status)
		}
	}
}

func TestRunRefs_URLRefWithURLField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	refs := []spec.Ref{
		{Kind: "url", URL: server.URL, Required: true},
	}
	results := RunRefs(refs)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "pass" {
		t.Errorf("expected pass, got %s: %s", results[0].Status, results[0].Error)
	}
}
