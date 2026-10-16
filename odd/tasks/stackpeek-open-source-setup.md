# Stackpeek open-source repository setup

## Objective
Add the missing reader-facing documentation, repeatable CI, and a standard permissive license to Stackpeek, using the owner-delegated choice rather than sending the choice back.

## Problem and rationale
The local repo has 14 tracked files, Go 1.22, a TUI entry point, detector tests, but no README, CI workflow, or license. The user asked to leave the public repositories properly set up. Matt and Ponytail recommend a concise, evidence-backed README and a single Go 1.22 CI job for format, vet, test, and build. The user later explicitly delegated license selection; this task applies the conservative MIT default recommended by the open-source skill and peers for a small Go utility, with the repository owner handle as the copyright notice.

## Scope authorized
- `README.md` — create concise usage, build/install, scan scope/limits, diagnostics, development and test instructions.
- `.github/workflows/ci.yml` — add one Go 1.22 workflow running format, vet, tests and build.
- `LICENSE` — MIT license for this small Go utility, with `fxckcode` as the copyright holder for 2026.
- `odd/tasks/stackpeek-open-source-setup.md`
- No push, PR, release, or GitHub API calls.

## Tasks
- [x] SP1 — Write an evidence-based README for a stranger using Stackpeek.
- [x] SP2 — Add CI for the project's declared Go version and real gates.
- [x] SP3 — Add the standard MIT LICENSE selected under the owner's delegation.

## Acceptance criteria
- README states the verified executable invocation, installation/build steps, actual detector scope, and the 50,000-entry / 64-level Terraform limits with incomplete-scan diagnostics. It makes no unsupported completeness, performance, security, or production-readiness claims.
- README links to the actual MIT `LICENSE` file; the license includes a copyright holder/year and full standard terms.
- CI uses Go 1.22 and runs `gofmt` check, `go vet ./...`, `go test ./...`, and `go build ./...`.
- YAML parses and the Go gates pass locally with network disabled.
- No non-standard license terms are invented; include the complete standard MIT text.

## Routing and risk
- SP1 + SP2: one delegated writer because this is a two-file doc/workflow setup; operational workflow risk is medium. Writer self-verifies, parent performs structural/YAML/readback and gate spot-checks.
- SP3: parent applies the user-delegated MIT default and verifies the complete license text.
- Forecast: approximately 100 authored lines; local-only, no delivery strategy beyond the existing feature branch.

## Verification / progress
- Base includes the verified Terraform scan change on `fxckcode/bounded-terraform-scan`, commit `a8fc26f5230b842da3e22f6a0db4994c47cc32e3`; branch was clean before packaging changes.
- The user directed the agent to choose the remaining controlled setup decisions. MIT is selected as the conservative default for this small Go utility, as recommended by the `open-sourcing-projects` skill and both peers; copyright notice uses repository owner handle `fxckcode` and year 2026.
- Matt and Ponytail recommended the minimal README and Go 1.22 CI gates; their MIT recommendation was conditional on owner delegation, now provided.
- CodeGraph is unavailable; no project-specific `.atl/skill-registry.md` exists. Engram mirror remains pending because it does not resolve Stackpeek.
- Parent read back the README and workflow; content matches the local entrypoint and detector behavior. README now links to the selected MIT LICENSE.
- Offline `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...` passed for all three packages; `git diff --check` passed. Workflow YAML parsed locally with `CPAN::Meta::YAML`; `actionlint` and PyYAML were unavailable and not installed.
- SP1/SP2 work-unit commit: `c00b94dc2c64931b53ac3ba0d9e96d057f2d3d01` (`docs: add Stackpeek README and Go CI`).
- SP3 MIT LICENSE added with copyright `fxckcode` (2026), linked from README; its text matches the full standard MIT grant. `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...`, local license comparison, and `git diff --check` passed.
- Next: commit SP3 locally. No push/PR/release authorized.
