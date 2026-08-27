package detector

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// DefaultMaxReadBytes is the default upper bound for optional file reads.
const DefaultMaxReadBytes int64 = 1 << 20

// ValidateRoot resolves path and ensures it is an existing readable directory.
// Missing, empty, inaccessible, or non-directory paths are fatal errors.
func ValidateRoot(ctx context.Context, path string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if path == "" {
		return "", fmt.Errorf("root path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve root %q: %w", path, err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", fmt.Errorf("stat root %q: %w", abs, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("root %q is not a directory", abs)
	}
	return abs, nil
}

// OpenRoot validates path and returns an fs.FS rooted at the resolved directory.
func OpenRoot(ctx context.Context, path string) (fs.FS, error) {
	abs, err := ValidateRoot(ctx, path)
	if err != nil {
		return nil, err
	}
	return os.DirFS(abs), nil
}

// ReadFileBounded reads at most maxBytes from name.
// Optional read/stat failures become diagnostics; context cancellation is fatal.
func ReadFileBounded(ctx context.Context, fsys fs.FS, name string, maxBytes int64) ([]byte, []Diagnostic, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxReadBytes
	}
	f, err := fsys.Open(name)
	if err != nil {
		return nil, []Diagnostic{{Path: name, Code: "read", Message: err.Error()}}, nil
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, maxBytes+1))
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, nil, ctxErr
		}
		return nil, []Diagnostic{{Path: name, Code: "read", Message: err.Error()}}, nil
	}
	if int64(len(data)) > maxBytes {
		msg := fmt.Sprintf("file exceeds %d byte read limit", maxBytes)
		return data[:maxBytes], []Diagnostic{{Path: name, Code: "truncated", Message: msg}}, nil
	}
	return data, nil, nil
}
