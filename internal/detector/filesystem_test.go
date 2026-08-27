package detector_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"stackpeek/internal/detector"
)

func TestValidateRoot_FatalCases(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	missing := filepath.Join(t.TempDir(), "missing")
	filePath := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ name, path string }{
		{"missing", missing},
		{"non-directory", filePath},
		{"empty", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := detector.ValidateRoot(ctx, tc.path); err == nil {
				t.Fatal("expected fatal error")
			}
		})
	}
}

func TestValidateRoot_AcceptsDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	got, err := detector.ValidateRoot(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if abs, _ := filepath.Abs(dir); got != dir && got != abs {
		t.Fatalf("got %q want %q", got, dir)
	}
}

func TestReadFileBounded_OptionalFailuresBecomeDiagnostics(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	root, err := detector.OpenRoot(ctx, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, diags, err := detector.ReadFileBounded(ctx, root, "missing.json", detector.DefaultMaxReadBytes)
	if err != nil || len(data) != 0 || len(diags) != 1 {
		t.Fatalf("data=%q diags=%v err=%v", data, diags, err)
	}
	if diags[0].Path != "missing.json" || diags[0].Code == "" || diags[0].Message == "" {
		t.Fatalf("bad diagnostic: %+v", diags[0])
	}
}

func TestReadFileBounded_ReadsWithinLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	want := []byte("hello-stackpeek")
	if err := os.WriteFile(filepath.Join(dir, "ok.txt"), want, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	root, err := detector.OpenRoot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, diags, err := detector.ReadFileBounded(ctx, root, "ok.txt", detector.DefaultMaxReadBytes)
	if err != nil || len(diags) != 0 || string(data) != string(want) {
		t.Fatalf("data=%q diags=%v err=%v", data, diags, err)
	}
}

func TestReadFileBounded_TruncatesOverLimit(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte("abcdefghij"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	root, err := detector.OpenRoot(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	data, diags, err := detector.ReadFileBounded(ctx, root, "big.txt", 4)
	if err != nil || string(data) != "abcd" || len(diags) != 1 || diags[0].Code != "truncated" {
		t.Fatalf("data=%q diags=%v err=%v", data, diags, err)
	}
}

func TestValidateRoot_CanceledContext(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := detector.ValidateRoot(ctx, t.TempDir()); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v want canceled", err)
	}
}
