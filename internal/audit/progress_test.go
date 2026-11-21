package audit

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/aschepis/maslow/internal/progress"
	"github.com/aschepis/maslow/internal/spec"
)

func TestRun_EmitsProgressPerTarget(t *testing.T) {
	binPath := filepath.Join(t.TempDir(), "test-bin")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho v1"), 0o755); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	mas := &spec.MAS{Audit: &spec.Audit{Targets: []spec.AuditTarget{
		{Kind: "binary", Path: binPath},
		{Kind: "binary", Path: missing},
	}}}

	type recorded struct {
		Phase    progress.Phase
		Total    int
		Kind     string
		Name     string
		Statuses []string
	}
	var got []recorded
	opts := Options{Profile: "full", Progress: func(e progress.Event) {
		got = append(got, recorded{e.Phase, e.Total, e.Kind, e.Name, e.Statuses})
	}}
	if _, err := Run(mas, opts); err != nil {
		t.Fatal(err)
	}

	want := []recorded{
		{Phase: progress.Plan, Total: 2},
		{Phase: progress.Start, Kind: "binary", Name: binPath},
		{Phase: progress.Finish, Kind: "binary", Name: binPath, Statuses: []string{"pass"}},
		{Phase: progress.Start, Kind: "binary", Name: missing},
		{Phase: progress.Finish, Kind: "binary", Name: missing, Statuses: []string{"fail"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events:\n got %+v\nwant %+v", got, want)
	}
}
