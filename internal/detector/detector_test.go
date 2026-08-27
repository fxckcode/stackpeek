package detector_test

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"stackpeek/internal/detector"
)

type stubDetector struct {
	name        string
	findings    []detector.Finding
	diagnostics []detector.Diagnostic
}

func (s stubDetector) Name() string { return s.name }
func (s stubDetector) Detect(context.Context, fs.FS) ([]detector.Finding, []detector.Diagnostic) {
	return append([]detector.Finding(nil), s.findings...), append([]detector.Diagnostic(nil), s.diagnostics...)
}

func TestAggregateFindings_DeduplicatesAndSorts(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []detector.Finding
		want []detector.Finding
	}{
		{
			"dedupe identical",
			[]detector.Finding{
				{Technology: "Node.js", Signal: "package.json"},
				{Technology: "Docker", Signal: "Dockerfile"},
				{Technology: "Node.js", Signal: "package.json"},
			},
			[]detector.Finding{
				{Technology: "Docker", Signal: "Dockerfile"},
				{Technology: "Node.js", Signal: "package.json"},
			},
		},
		{
			"same tech different signals",
			[]detector.Finding{
				{Technology: "PHP", Signal: "composer.json"},
				{Technology: "PHP", Signal: "artisan"},
			},
			[]detector.Finding{
				{Technology: "PHP", Signal: "artisan"},
				{Technology: "PHP", Signal: "composer.json"},
			},
		},
		{
			"sort technology then signal",
			[]detector.Finding{
				{Technology: "Terraform", Signal: "main.tf"},
				{Technology: "Docker", Signal: "Dockerfile"},
				{Technology: "Docker Compose", Signal: "docker-compose.yml"},
			},
			[]detector.Finding{
				{Technology: "Docker", Signal: "Dockerfile"},
				{Technology: "Docker Compose", Signal: "docker-compose.yml"},
				{Technology: "Terraform", Signal: "main.tf"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			eqFindings(t, detector.AggregateFindings(tc.in), tc.want)
		})
	}
}

func TestAggregateDiagnostics_SortsByPathCodeMessage(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   []detector.Diagnostic
		want []detector.Diagnostic
	}{
		{
			"path code message",
			[]detector.Diagnostic{
				{Path: "b.json", Code: "parse", Message: "z"},
				{Path: "a.json", Code: "read", Message: "x"},
				{Path: "a.json", Code: "parse", Message: "y"},
				{Path: "a.json", Code: "parse", Message: "a"},
			},
			[]detector.Diagnostic{
				{Path: "a.json", Code: "parse", Message: "a"},
				{Path: "a.json", Code: "parse", Message: "y"},
				{Path: "a.json", Code: "read", Message: "x"},
				{Path: "b.json", Code: "parse", Message: "z"},
			},
		},
		{"empty", []detector.Diagnostic{}, []detector.Diagnostic{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			eqDiags(t, detector.AggregateDiagnostics(tc.in), tc.want)
		})
	}
}

func TestRegistryScan_RegistrationOrderIndependent(t *testing.T) {
	t.Parallel()
	a := stubDetector{
		name: "a",
		findings: []detector.Finding{
			{Technology: "Node.js", Signal: "package.json"},
		},
		diagnostics: []detector.Diagnostic{{Path: "pkg", Code: "warn", Message: "a"}},
	}
	b := stubDetector{
		name: "b",
		findings: []detector.Finding{
			{Technology: "Docker", Signal: "Dockerfile"},
			{Technology: "Node.js", Signal: "package.json"},
		},
		diagnostics: []detector.Diagnostic{{Path: "docker", Code: "warn", Message: "b"}},
	}
	ctx, root := context.Background(), fstest.MapFS{}
	first := detector.NewRegistry(a, b).Scan(ctx, ".", root)
	second := detector.NewRegistry(b, a).Scan(ctx, ".", root)
	eqFindings(t, first.Findings, second.Findings)
	eqDiags(t, first.Diagnostics, second.Diagnostics)
	eqFindings(t, first.Findings, []detector.Finding{
		{Technology: "Docker", Signal: "Dockerfile"},
		{Technology: "Node.js", Signal: "package.json"},
	})
	eqDiags(t, first.Diagnostics, []detector.Diagnostic{
		{Path: "docker", Code: "warn", Message: "b"},
		{Path: "pkg", Code: "warn", Message: "a"},
	})
	if first.Root != "." || first.Fatal != nil {
		t.Fatalf("Root=%q Fatal=%v", first.Root, first.Fatal)
	}
}

func TestRegistryScan_DoesNotMutateCallerSlices(t *testing.T) {
	t.Parallel()
	findings := []detector.Finding{{Technology: "Zed", Signal: "z"}, {Technology: "Alpha", Signal: "a"}}
	diags := []detector.Diagnostic{{Path: "z", Code: "c", Message: "m"}}
	origF, origD := append([]detector.Finding(nil), findings...), append([]detector.Diagnostic(nil), diags...)
	_ = detector.NewRegistry(stubDetector{name: "s", findings: findings, diagnostics: diags}).
		Scan(context.Background(), ".", fstest.MapFS{})
	eqFindings(t, findings, origF)
	eqDiags(t, diags, origD)
}

func TestAggregateFindings_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	in := []detector.Finding{{Technology: "B", Signal: "2"}, {Technology: "A", Signal: "1"}}
	orig := append([]detector.Finding(nil), in...)
	_ = detector.AggregateFindings(in)
	eqFindings(t, in, orig)
}

func eqFindings(t *testing.T, got, want []detector.Finding) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d]=%+v want=%+v", i, got[i], want[i])
		}
	}
}

func eqDiags(t *testing.T, got, want []detector.Diagnostic) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len=%d want=%d got=%v want=%v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d]=%+v want=%+v", i, got[i], want[i])
		}
	}
}
