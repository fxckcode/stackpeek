package detector

import (
	"context"
	"encoding/json"
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

type terraformDetector struct{}

func (terraformDetector) Name() string { return "terraform" }
func (terraformDetector) Detect(ctx context.Context, root fs.FS) ([]Finding, []Diagnostic) {
	var out []Finding
	_ = fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
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
	return out, nil
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
