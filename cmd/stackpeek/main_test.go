package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"stackpeek/internal/tui"
)

func writeFixture(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRun_DefaultPathUsesCWD(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "package.json", `{"name":"x"}`)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	var got tui.Model
	var stderr bytes.Buffer
	if code := run(nil, &bytes.Buffer{}, &stderr, func(m tui.Model) error { got = m; return nil }); code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if v := got.View(); !strings.Contains(v, "Node.js") || !strings.Contains(v, dir) {
		t.Fatalf("missing Node.js/root:\n%s", v)
	}
}

func TestRun_ExplicitPath(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "Dockerfile", "FROM scratch\n")
	var got tui.Model
	var stderr bytes.Buffer
	if code := run([]string{dir}, &bytes.Buffer{}, &stderr, func(m tui.Model) error { got = m; return nil }); code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	if v := got.View(); !strings.Contains(v, "Docker") || !strings.Contains(v, dir) {
		t.Fatalf("missing Docker/root:\n%s", v)
	}
}

func TestRun_InvalidPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	var launched bool
	var stderr bytes.Buffer
	code := run([]string{missing}, &bytes.Buffer{}, &stderr, func(tui.Model) error { launched = true; return nil })
	if code == 0 || launched {
		t.Fatalf("want non-zero no-launch; code=%d launched=%v", code, launched)
	}
	if e := stderr.String(); !strings.Contains(e, missing) && !strings.Contains(e, "stat root") {
		t.Fatalf("stderr not actionable: %q", e)
	}
}

func TestRun_DoesNotMutateFixture(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "package.json")
	want := []byte(`{"name":"keep"}`)
	if err := os.WriteFile(path, want, 0o644); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	if code := run([]string{dir}, &bytes.Buffer{}, &stderr, func(tui.Model) error { return nil }); code != 0 {
		t.Fatalf("exit %d stderr=%q", code, stderr.String())
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("fixture mutated or unreadable: err=%v got=%q", err, got)
	}
}
