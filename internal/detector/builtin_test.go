package detector_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"stackpeek/internal/detector"
)

func TestBuiltinSignals_ExactMatchAndUnrelatedAbsent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name, absent string
		files        map[string]string
		want         []detector.Finding
	}{
		{"package.json Node", "NestJS,PHP,Python,Docker,Django,Laravel",
			map[string]string{"package.json": `{"name":"app"}`},
			[]detector.Finding{{Technology: "Node.js", Signal: "package.json"}}},
		{"composer.json PHP", "Laravel,Node.js,Python",
			map[string]string{"composer.json": `{"name":"app/pkg"}`},
			[]detector.Finding{{Technology: "PHP", Signal: "composer.json"}}},
		{"requirements.txt Python", "Django,Node.js",
			map[string]string{"requirements.txt": "requests==2.0.0\n"},
			[]detector.Finding{{Technology: "Python", Signal: "requirements.txt"}}},
		{"Dockerfile", "Docker Compose,Node.js",
			map[string]string{"Dockerfile": "FROM alpine\n"},
			[]detector.Finding{{Technology: "Docker", Signal: "Dockerfile"}}},
		{"docker-compose.yml", "Docker,Terraform",
			map[string]string{"docker-compose.yml": "services:\n  web:\n    image: nginx\n"},
			[]detector.Finding{{Technology: "Docker Compose", Signal: "docker-compose.yml"}}},
		{"Terraform", "Docker,GitHub Actions",
			map[string]string{"main.tf": `resource "null_resource" "x" {}`},
			[]detector.Finding{{Technology: "Terraform", Signal: "main.tf"}}},
		{"GitHub Actions", "Terraform,Docker",
			map[string]string{".github/workflows/ci.yml": "name: ci\non: push\n"},
			[]detector.Finding{{Technology: "GitHub Actions", Signal: ".github/workflows/ci.yml"}}},
		{".env.example", "Node.js,Docker",
			map[string]string{".env.example": "APP_ENV=local\n"},
			[]detector.Finding{{Technology: "Environment Template", Signal: ".env.example"}}},
		{"Prisma", "NestJS,Django",
			map[string]string{"prisma/schema.prisma": "generator client {\n  provider = \"prisma-client-js\"\n}\n"},
			[]detector.Finding{{Technology: "Prisma", Signal: "prisma/schema.prisma"}}},
		{"Laravel artisan", "NestJS,Django,Node.js",
			map[string]string{"artisan": "#!/usr/bin/env php\n"},
			[]detector.Finding{{Technology: "Laravel", Signal: "artisan"}}},
		{"Laravel composer signal", "NestJS,Django",
			map[string]string{"composer.json": `{"require":{"laravel/framework":"^11.0"}}`},
			[]detector.Finding{
				{Technology: "Laravel", Signal: "composer.json"},
				{Technology: "PHP", Signal: "composer.json"},
			}},
		{"NestJS dependency", "Laravel,Django,Prisma",
			map[string]string{"package.json": `{"dependencies":{"@nestjs/core":"10.0.0"}}`},
			[]detector.Finding{
				{Technology: "NestJS", Signal: "package.json"},
				{Technology: "Node.js", Signal: "package.json"},
			}},
		{"Django dependency", "NestJS,Laravel,Node.js",
			map[string]string{"requirements.txt": "Django==5.0\npsycopg2==2.9\n"},
			[]detector.Finding{
				{Technology: "Django", Signal: "requirements.txt"},
				{Technology: "Python", Signal: "requirements.txt"},
			}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := scanDefault(t, writeFixture(t, tc.files))
			eqFindings(t, got.Findings, tc.want)
			for _, tech := range splitCSV(tc.absent) {
				assertTechnologyAbsent(t, got.Findings, tech)
			}
			assertSourcesProvenance(t, got.Findings)
			if got.Fatal != nil {
				t.Fatalf("Fatal=%v", got.Fatal)
			}
		})
	}
}

func TestBuiltin_MalformedPackageAndDependencyDiagnostics(t *testing.T) {
	t.Parallel()
	got := scanDefault(t, writeFixture(t, map[string]string{
		"package.json": `{not-json`, "composer.json": `{"require":`,
		"requirements.txt": "Django==5.0\n", "Dockerfile": "FROM alpine\n",
	}))
	eqFindings(t, got.Findings, []detector.Finding{
		{Technology: "Django", Signal: "requirements.txt"},
		{Technology: "Docker", Signal: "Dockerfile"},
		{Technology: "Python", Signal: "requirements.txt"},
	})
	assertHasDiag(t, got.Diagnostics, "package.json", "parse")
	assertHasDiag(t, got.Diagnostics, "composer.json", "parse")
	for _, tech := range []string{"Node.js", "PHP", "NestJS", "Laravel"} {
		assertTechnologyAbsent(t, got.Findings, tech)
	}
	if got.Fatal != nil {
		t.Fatalf("Fatal=%v", got.Fatal)
	}
}

func TestDefaultRegistry_SourceProvenanceStable(t *testing.T) {
	t.Parallel()
	got := scanDefault(t, writeFixture(t, map[string]string{
		"package.json":                  `{"dependencies":{"@nestjs/core":"1.0.0"}}`,
		"composer.json":                 `{"require":{"laravel/framework":"^10"}}`,
		"requirements.txt":              "Django>=4\n",
		"Dockerfile":                    "FROM scratch\n",
		"docker-compose.yml":            "services: {}\n",
		"infra/main.tf":                 `resource "null_resource" "x" {}`,
		".github/workflows/deploy.yaml": "name: d\n",
		".env.example":                  "X=1\n",
		"prisma/schema.prisma":          "model User { id Int @id }\n",
		"artisan":                       "#!/usr/bin/env php\n",
	}))
	eqFindings(t, got.Findings, []detector.Finding{
		{Technology: "Django", Signal: "requirements.txt"},
		{Technology: "Docker", Signal: "Dockerfile"},
		{Technology: "Docker Compose", Signal: "docker-compose.yml"},
		{Technology: "Environment Template", Signal: ".env.example"},
		{Technology: "GitHub Actions", Signal: ".github/workflows/deploy.yaml"},
		{Technology: "Laravel", Signal: "artisan"},
		{Technology: "Laravel", Signal: "composer.json"},
		{Technology: "NestJS", Signal: "package.json"},
		{Technology: "Node.js", Signal: "package.json"},
		{Technology: "PHP", Signal: "composer.json"},
		{Technology: "Prisma", Signal: "prisma/schema.prisma"},
		{Technology: "Python", Signal: "requirements.txt"},
		{Technology: "Terraform", Signal: "infra/main.tf"},
	})
	assertSourcesProvenance(t, got.Findings)
	if len(got.Diagnostics) != 0 {
		t.Fatalf("unexpected diagnostics: %v", got.Diagnostics)
	}
}

func TestCanceledBuiltinScanIsNotClean(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got := detector.DefaultRegistry().Scan(ctx, ".", fstest.MapFS{})
	if !errors.Is(got.Fatal, context.Canceled) {
		t.Fatalf("expected context cancellation to be fatal, got %+v", got)
	}
}

func TestTerraformEntryLimitBoundaries(t *testing.T) {
	cases := []struct {
		name          string
		fileCount     int
		wantFindings  int
		wantLimitDiag bool
	}{
		{name: "below limit", fileCount: 49_998, wantFindings: 49_998},
		{name: "at limit", fileCount: 49_999, wantFindings: 49_999},
		{name: "over limit", fileCount: 50_000, wantFindings: 49_999, wantLimitDiag: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := make(fstest.MapFS, tc.fileCount)
			for i := 0; i < tc.fileCount; i++ {
				name := fmt.Sprintf("file-%05d.tf", i)
				files[name] = &fstest.MapFile{Data: []byte("resource {}")}
			}

			got := detector.DefaultRegistry().Scan(context.Background(), ".", files)
			terraformFindings := 0
			for _, finding := range got.Findings {
				if finding.Technology == detector.TechTerraform {
					terraformFindings++
				}
			}
			if terraformFindings != tc.wantFindings {
				t.Fatalf("Terraform findings = %d, want %d", terraformFindings, tc.wantFindings)
			}
			if hasIncompleteDiagCode(got.Diagnostics, "limit") != tc.wantLimitDiag {
				t.Fatalf("entry limit diagnostic presence = %t, want %t; diagnostics: %v", hasIncompleteDiagCode(got.Diagnostics, "limit"), tc.wantLimitDiag, got.Diagnostics)
			}
			if got.Fatal != nil {
				t.Fatalf("Fatal=%v", got.Fatal)
			}
		})
	}
}

func TestTerraformDepthLimitBoundaries(t *testing.T) {
	cases := []struct {
		name          string
		depth         int
		wantFinding   bool
		wantLimitDiag bool
	}{
		{name: "below limit", depth: 63, wantFinding: true},
		{name: "at limit", depth: 64, wantFinding: true},
		{name: "over limit", depth: 65, wantLimitDiag: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			deepPath := strings.Repeat("level/", tc.depth-1) + "main.tf"
			files := fstest.MapFS{
				"a.tf":   &fstest.MapFile{Data: []byte("resource {}")},
				deepPath: &fstest.MapFile{Data: []byte("resource {}")},
			}

			got := detector.DefaultRegistry().Scan(context.Background(), ".", files)
			if containsFinding(got.Findings, detector.TechTerraform, deepPath) != tc.wantFinding {
				t.Fatalf("finding at depth %d present = %t, want %t", tc.depth, containsFinding(got.Findings, detector.TechTerraform, deepPath), tc.wantFinding)
			}
			if !containsFinding(got.Findings, detector.TechTerraform, "a.tf") {
				t.Fatalf("expected readable sibling finding before depth boundary, got %v", got.Findings)
			}
			if hasIncompleteDiagCode(got.Diagnostics, "limit") != tc.wantLimitDiag {
				t.Fatalf("depth limit diagnostic presence = %t, want %t; diagnostics: %v", hasIncompleteDiagCode(got.Diagnostics, "limit"), tc.wantLimitDiag, got.Diagnostics)
			}
			if got.Fatal != nil {
				t.Fatalf("Fatal=%v", got.Fatal)
			}
		})
	}
}

func TestTerraformCancellationDuringTraversalIsNotClean(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	root := &cancelOnOpenFS{
		FS:     fstest.MapFS{"a.tf": &fstest.MapFile{Data: []byte("resource {}")}, "z/b.tf": &fstest.MapFile{Data: []byte("resource {}")}},
		name:   "z",
		cancel: cancel,
	}

	got := detector.DefaultRegistry().Scan(ctx, ".", root)
	if !root.opened {
		t.Fatal("expected traversal to open the nested directory before cancellation")
	}
	if !errors.Is(got.Fatal, context.Canceled) {
		t.Fatalf("expected in-flight cancellation to be fatal, got %+v", got)
	}
}

func TestTerraformWalkErrorReportsIncompleteResults(t *testing.T) {
	base := fstest.MapFS{
		"main.tf":          &fstest.MapFile{Data: []byte("resource {}")},
		"broken/hidden.tf": &fstest.MapFile{Data: []byte("resource {}")},
	}
	root := walkErrorFS{FS: base, path: "broken", err: fs.ErrPermission}
	got := detector.DefaultRegistry().Scan(context.Background(), ".", root)

	if !containsFinding(got.Findings, detector.TechTerraform, "main.tf") {
		t.Fatalf("expected finding outside unreadable directory, got %v", got.Findings)
	}
	assertHasDiag(t, got.Diagnostics, "broken", "walk")
	if got.Fatal != nil {
		t.Fatalf("Fatal=%v", got.Fatal)
	}
}

type walkErrorFS struct {
	fs.FS
	path string
	err  error
}

func (f walkErrorFS) Open(name string) (fs.File, error) {
	if name == f.path {
		return nil, f.err
	}
	return f.FS.Open(name)
}

type cancelOnOpenFS struct {
	fs.FS
	name   string
	cancel context.CancelFunc
	opened bool
}

func (f *cancelOnOpenFS) Open(name string) (fs.File, error) {
	if name == f.name && !f.opened {
		f.opened = true
		f.cancel()
	}
	return f.FS.Open(name)
}

func containsFinding(findings []detector.Finding, technology, signal string) bool {
	for _, finding := range findings {
		if finding.Technology == technology && finding.Signal == signal {
			return true
		}
	}
	return false
}

func writeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func scanDefault(t *testing.T, rootPath string) detector.ScanResult {
	t.Helper()
	ctx := context.Background()
	root, err := detector.OpenRoot(ctx, rootPath)
	if err != nil {
		t.Fatal(err)
	}
	return detector.DefaultRegistry().Scan(ctx, rootPath, root)
}

func assertTechnologyAbsent(t *testing.T, findings []detector.Finding, tech string) {
	t.Helper()
	for _, f := range findings {
		if f.Technology == tech {
			t.Fatalf("technology %q should be absent; got %v", tech, findings)
		}
	}
}

func assertSourcesProvenance(t *testing.T, findings []detector.Finding) {
	t.Helper()
	if len(findings) == 0 {
		t.Fatal("expected findings with provenance")
	}
	for _, f := range findings {
		if f.Technology == "" || f.Signal == "" {
			t.Fatalf("missing provenance: %+v", f)
		}
	}
}

func assertHasDiagCode(t *testing.T, diags []detector.Diagnostic, code string) {
	t.Helper()
	if !hasIncompleteDiagCode(diags, code) {
		t.Fatalf("missing incomplete diagnostic code=%q in %v", code, diags)
	}
}

func hasIncompleteDiagCode(diags []detector.Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code && strings.Contains(strings.ToLower(d.Message), "incomplete") {
			return true
		}
	}
	return false
}

func assertHasDiag(t *testing.T, diags []detector.Diagnostic, path, code string) {
	t.Helper()
	for _, d := range diags {
		if d.Path == path && d.Code == code && d.Message != "" {
			return
		}
	}
	t.Fatalf("missing diagnostic path=%q code=%q in %v", path, code, diags)
}

func splitCSV(s string) []string { return strings.Split(s, ",") }
