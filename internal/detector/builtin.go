package detector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Stable technology names and primary signal paths for built-in detectors.
const (
	TechNodeJS, TechPHP, TechPython                = "Node.js", "PHP", "Python"
	TechDocker, TechDockerCompose, TechTerraform   = "Docker", "Docker Compose", "Terraform"
	TechGitHubActions, TechEnvTemplate, TechPrisma = "GitHub Actions", "Environment Template", "Prisma"
	TechLaravel, TechNestJS, TechDjango            = "Laravel", "NestJS", "Django"
	SignalPackageJSON, SignalComposerJSON          = "package.json", "composer.json"
	SignalRequirementsTxt, SignalDockerfile        = "requirements.txt", "Dockerfile"
	SignalDockerComposeYML, SignalEnvExample       = "docker-compose.yml", ".env.example"
	SignalPrismaSchema, SignalArtisan              = "prisma/schema.prisma", "artisan"
)

// DefaultRegistry returns built-in read-only detectors (bounded FS reads only; no mutation/secrets/network/exec).
func DefaultRegistry() Registry {
	return NewRegistry(
		filePresenceDetector{"docker", TechDocker, SignalDockerfile},
		filePresenceDetector{"compose", TechDockerCompose, SignalDockerComposeYML},
		filePresenceDetector{"env-template", TechEnvTemplate, SignalEnvExample},
		filePresenceDetector{"prisma", TechPrisma, SignalPrismaSchema},
		filePresenceDetector{"laravel-artisan", TechLaravel, SignalArtisan},
		packageJSONDetector{}, composerJSONDetector{}, requirementsDetector{},
		terraformDetector{}, githubActionsDetector{},
	)
}

type filePresenceDetector struct{ name, technology, file string }

func (d filePresenceDetector) Name() string { return d.name }
func (d filePresenceDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	if ctx.Err() != nil {
		return nil, nil
	}
	if _, err := fs.Stat(root, d.file); err != nil {
		return nil, nil
	}
	return []Finding{{Technology: d.technology, Signal: d.file}}, nil
}

type packageJSONDetector struct{}

func (packageJSONDetector) Name() string { return "package-json" }
func (packageJSONDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	data, diags, ok := readOptional(ctx, root, SignalPackageJSON)
	if !ok {
		return nil, diags
	}
	var doc struct {
		Dependencies, DevDependencies map[string]string
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, append(diags, Diagnostic{Path: SignalPackageJSON, Code: "parse", Message: err.Error()})
	}
	out := []Finding{{Technology: TechNodeJS, Signal: SignalPackageJSON}}
	if hasAnyDep(doc.Dependencies, doc.DevDependencies, "@nestjs/core", "@nestjs/common") {
		out = append(out, Finding{Technology: TechNestJS, Signal: SignalPackageJSON})
	}
	return out, diags
}

type composerJSONDetector struct{}

func (composerJSONDetector) Name() string { return "composer-json" }
func (composerJSONDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	data, diags, ok := readOptional(ctx, root, SignalComposerJSON)
	if !ok {
		return nil, diags
	}
	var doc struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, append(diags, Diagnostic{Path: SignalComposerJSON, Code: "parse", Message: err.Error()})
	}
	out := []Finding{{Technology: TechPHP, Signal: SignalComposerJSON}}
	if hasAnyDep(doc.Require, doc.RequireDev, "laravel/framework") {
		out = append(out, Finding{Technology: TechLaravel, Signal: SignalComposerJSON})
	}
	return out, diags
}

type requirementsDetector struct{}

func (requirementsDetector) Name() string { return "requirements-txt" }
func (requirementsDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	data, diags, ok := readOptional(ctx, root, SignalRequirementsTxt)
	if !ok {
		return nil, diags
	}
	out := []Finding{{Technology: TechPython, Signal: SignalRequirementsTxt}}
	if requirementsHasPackage(string(data), "Django") {
		out = append(out, Finding{Technology: TechDjango, Signal: SignalRequirementsTxt})
	}
	return out, diags
}

const (
	terraformMaxEntries = 50_000
	terraformMaxDepth   = 64
)

var errTerraformTraversalLimit = errors.New("Terraform traversal limit reached")

type terraformDetector struct{}

func (terraformDetector) Name() string { return "terraform" }
func (terraformDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	var out []Finding
	var diags []Diagnostic
	entries := 0
	walkErr := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			diags = append(diags, Diagnostic{Path: p, Code: "cancelled", Message: "Terraform scan is incomplete: context canceled"})
			return ctx.Err()
		}
		if entries >= terraformMaxEntries {
			diags = append(diags, Diagnostic{
				Path:    p,
				Code:    "limit",
				Message: fmt.Sprintf("Terraform scan is incomplete: entry limit of %d reached", terraformMaxEntries),
			})
			return errTerraformTraversalLimit
		}
		entries++
		if err != nil {
			diags = append(diags, Diagnostic{Path: p, Code: "walk", Message: fmt.Sprintf("Terraform scan is incomplete: %v", err)})
			return nil
		}
		depth := 0
		if p != "." {
			depth = strings.Count(p, "/") + 1
		}
		if depth > terraformMaxDepth {
			diags = append(diags, Diagnostic{
				Path:    p,
				Code:    "limit",
				Message: fmt.Sprintf("Terraform scan is incomplete: path depth limit of %d reached", terraformMaxDepth),
			})
			return errTerraformTraversalLimit
		}
		if d.IsDir() {
			switch path.Base(p) {
			case "node_modules", ".git", "vendor":
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(p, ".tf") {
			out = append(out, Finding{Technology: TechTerraform, Signal: p})
		}
		return nil
	})
	if walkErr != nil && !errors.Is(walkErr, errTerraformTraversalLimit) && !errors.Is(walkErr, ctx.Err()) {
		diags = append(diags, Diagnostic{Path: ".", Code: "walk", Message: fmt.Sprintf("Terraform scan is incomplete: %v", walkErr)})
	}
	return out, diags
}

type githubActionsDetector struct{}

func (githubActionsDetector) Name() string { return "github-actions" }
func (githubActionsDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	if ctx.Err() != nil {
		return nil, nil
	}
	entries, err := fs.ReadDir(root, ".github/workflows")
	if err != nil {
		return nil, nil
	}
	var out []Finding
	for _, e := range entries {
		if ctx.Err() != nil {
			return out, nil
		}
		if e.IsDir() {
			continue
		}
		if ext := path.Ext(e.Name()); ext == ".yml" || ext == ".yaml" {
			out = append(out, Finding{Technology: TechGitHubActions, Signal: path.Join(".github/workflows", e.Name())})
		}
	}
	return out, nil
}

func hasAnyDep(a, b map[string]string, names ...string) bool {
	for _, n := range names {
		if hasDep(a, n) || hasDep(b, n) {
			return true
		}
	}
	return false
}
func hasDep(m map[string]string, name string) bool { _, ok := m[name]; return ok }

func requirementsHasPackage(content, pkg string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.FieldsFunc(line, func(r rune) bool {
			return r == '=' || r == '<' || r == '>' || r == '!' || r == '~' || r == '[' || r == ' '
		})
		if len(parts) > 0 && strings.EqualFold(parts[0], pkg) {
			return true
		}
	}
	return false
}

// readOptional returns bounded bytes when present; missing files are silent; ctx errors yield !ok.
func readOptional(ctx context.Context, root fs.FS, name string) ([]byte, []Diagnostic, bool) {
	if ctx.Err() != nil {
		return nil, nil, false
	}
	if _, err := fs.Stat(root, name); err != nil {
		return nil, nil, false
	}
	data, diags, err := ReadFileBounded(ctx, root, name, DefaultMaxReadBytes)
	if err != nil || len(data) == 0 {
		return nil, diags, false
	}
	return data, diags, true
}
