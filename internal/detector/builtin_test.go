package detector_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
