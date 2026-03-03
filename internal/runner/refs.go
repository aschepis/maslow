package runner

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aschepis/maslow-agentic/internal/evidence"
	"github.com/aschepis/maslow-agentic/internal/spec"
)

// RunRefs verifies all declared refs and returns results.
func RunRefs(refs []spec.Ref) []evidence.RefResult {
	var results []evidence.RefResult
	for _, ref := range refs {
		result := runRef(ref)
		results = append(results, result)
	}
	return results
}

func runRef(ref spec.Ref) evidence.RefResult {
	switch ref.Kind {
	case "file":
		return checkFileRef(ref)
	case "binary":
		return checkBinaryRef(ref)
	case "url":
		return checkURLRef(ref)
	case "package", "container", "endpoint", "mcp":
		return evidence.RefResult{
			Path:   refPath(ref),
			Kind:   ref.Kind,
			Status: "skip",
			Error:  fmt.Sprintf("%s ref verification not yet implemented", ref.Kind),
		}
	default:
		return evidence.RefResult{
			Path:   refPath(ref),
			Kind:   ref.Kind,
			Status: "skip",
			Error:  fmt.Sprintf("unknown ref kind: %s", ref.Kind),
		}
	}
}

func checkFileRef(ref spec.Ref) evidence.RefResult {
	_, err := os.Stat(ref.Path)
	if err != nil {
		if ref.Required {
			return evidence.RefResult{
				Path:   ref.Path,
				Kind:   ref.Kind,
				Status: "fail",
				Error:  fmt.Sprintf("required file ref not found: %s", ref.Path),
			}
		}
		return evidence.RefResult{
			Path:   ref.Path,
			Kind:   ref.Kind,
			Status: "skip",
			Error:  fmt.Sprintf("optional file ref not found: %s", ref.Path),
		}
	}
	return evidence.RefResult{
		Path:   ref.Path,
		Kind:   ref.Kind,
		Status: "pass",
	}
}

func checkBinaryRef(ref spec.Ref) evidence.RefResult {
	info, err := os.Stat(ref.Path)
	if err != nil {
		if ref.Required {
			return evidence.RefResult{
				Path:   ref.Path,
				Kind:   ref.Kind,
				Status: "fail",
				Error:  fmt.Sprintf("required binary ref not found: %s", ref.Path),
			}
		}
		return evidence.RefResult{
			Path:   ref.Path,
			Kind:   ref.Kind,
			Status: "skip",
			Error:  fmt.Sprintf("optional binary ref not found: %s", ref.Path),
		}
	}

	// Check executable bit.
	if info.Mode()&0o111 == 0 {
		return evidence.RefResult{
			Path:   ref.Path,
			Kind:   ref.Kind,
			Status: "fail",
			Error:  fmt.Sprintf("binary ref exists but is not executable: %s", ref.Path),
		}
	}

	return evidence.RefResult{
		Path:   ref.Path,
		Kind:   ref.Kind,
		Status: "pass",
	}
}

func checkURLRef(ref spec.Ref) evidence.RefResult {
	url := ref.Path
	if ref.URL != "" {
		url = ref.URL
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Head(url)
	if err != nil {
		if ref.Required {
			return evidence.RefResult{
				Path:   url,
				Kind:   ref.Kind,
				Status: "fail",
				Error:  fmt.Sprintf("required URL ref unreachable: %v", err),
			}
		}
		return evidence.RefResult{
			Path:   url,
			Kind:   ref.Kind,
			Status: "skip",
			Error:  fmt.Sprintf("optional URL ref unreachable: %v", err),
		}
	}
	resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return evidence.RefResult{
			Path:   url,
			Kind:   ref.Kind,
			Status: "pass",
		}
	}

	if ref.Required {
		return evidence.RefResult{
			Path:   url,
			Kind:   ref.Kind,
			Status: "fail",
			Error:  fmt.Sprintf("required URL ref returned status %d", resp.StatusCode),
		}
	}
	return evidence.RefResult{
		Path:   url,
		Kind:   ref.Kind,
		Status: "skip",
		Error:  fmt.Sprintf("optional URL ref returned status %d", resp.StatusCode),
	}
}

// refPath returns the best path/identifier for a ref.
func refPath(ref spec.Ref) string {
	if ref.Path != "" {
		return ref.Path
	}
	if ref.URL != "" {
		return ref.URL
	}
	if ref.Name != "" {
		return ref.Name
	}
	return "(unknown)"
}
