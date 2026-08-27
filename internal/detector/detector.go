// Package detector defines the scan contract, registry, and deterministic aggregation.
package detector

import (
	"context"
	"io/fs"
	"sort"
)

// Finding is a normalized technology signal discovered in a repository.
type Finding struct {
	Technology string
	Signal     string
}

// Diagnostic describes a non-fatal problem encountered while scanning.
type Diagnostic struct {
	Path    string
	Code    string
	Message string
}

// ScanResult is the immutable output of a registry scan.
type ScanResult struct {
	Root        string
	Findings    []Finding
	Diagnostics []Diagnostic
	Fatal       error
}

// Detector inspects a repository root and returns findings plus diagnostics.
type Detector interface {
	Name() string
	Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic)
}

// Registry holds detectors and runs them against a shared root.
type Registry struct{ detectors []Detector }

// NewRegistry copies detectors into an independent registry.
func NewRegistry(detectors ...Detector) Registry {
	cp := make([]Detector, len(detectors))
	copy(cp, detectors)
	return Registry{detectors: cp}
}

// Scan invokes every detector, then deduplicates and sorts results.
func (r Registry) Scan(ctx context.Context, rootPath string, root fs.FS) ScanResult {
	var findings []Finding
	var diags []Diagnostic
	for _, d := range r.detectors {
		if err := ctx.Err(); err != nil {
			return ScanResult{Root: rootPath, Fatal: err}
		}
		f, dg := d.Detect(ctx, root)
		findings = append(findings, f...)
		diags = append(diags, dg...)
	}
	return ScanResult{Root: rootPath, Findings: AggregateFindings(findings), Diagnostics: AggregateDiagnostics(diags)}
}

// AggregateFindings returns a sorted, deduplicated copy keyed by (Technology, Signal).
func AggregateFindings(in []Finding) []Finding {
	out := append([]Finding(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Technology != out[j].Technology {
			return out[i].Technology < out[j].Technology
		}
		return out[i].Signal < out[j].Signal
	})
	n := 0
	for _, f := range out {
		if n == 0 || out[n-1] != f {
			out[n] = f
			n++
		}
	}
	if n == 0 {
		return []Finding{}
	}
	return out[:n]
}

// AggregateDiagnostics returns a sorted copy by Path, Code, then Message.
func AggregateDiagnostics(in []Diagnostic) []Diagnostic {
	out := append([]Diagnostic(nil), in...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	if len(out) == 0 {
		return []Diagnostic{}
	}
	return out
}
