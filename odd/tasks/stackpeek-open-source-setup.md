# Stackpeek open-source repository setup

## Objective
Add the missing reader-facing documentation and repeatable CI to Stackpeek without inventing project claims or choosing legal terms for the owner.

## Problem and rationale
The local repo has 14 tracked files, Go 1.22, a TUI entry point, detector tests, but no README, CI workflow, or license. The user asked to leave the public repositories properly set up. Matt and Ponytail recommend a concise, evidence-backed README and a single Go 1.22 CI job for format, vet, test, and build. They both say the license remains the owner's legal decision; a clarification proposing MIT timed out, so do not add any LICENSE file or infer terms from Envy.

## Scope authorized
- `README.md` — create concise usage, build/install, scan scope/limits, diagnostics, development and test instructions.
- `.github/workflows/ci.yml` — add one Go 1.22 workflow running format, vet, tests and build.
- `odd/tasks/stackpeek-open-source-setup.md`
- `LICENSE` is explicitly not in scope until the owner selects license terms.
- No push, PR, release, or GitHub API calls.

## Tasks
- [x] SP1 — Write an evidence-based README for a stranger using Stackpeek.
- [x] SP2 — Add CI for the project's declared Go version and real gates.
- [ ] SP3 — Add a LICENSE only after the owner chooses terms; currently blocked by unanswered license clarification.

## Acceptance criteria
- README states the verified executable invocation, installation/build steps, actual detector scope, and the 50,000-entry / 64-level Terraform limits with incomplete-scan diagnostics. It makes no unsupported completeness, performance, security, or production-readiness claims.
- README includes a clear license-status section stating that no license has been selected; it must not imply Stackpeek is legally reusable.
- CI uses Go 1.22 and runs `gofmt` check, `go vet ./...`, `go test ./...`, and `go build ./...`.
- YAML parses and the Go gates pass locally with network disabled.
- No license terms are guessed, and no remote branch, push, PR, or release is created.

## Routing and risk
- SP1 + SP2: one delegated writer because this is a two-file doc/workflow setup; operational workflow risk is medium. Writer self-verifies, parent performs structural/YAML/readback and gate spot-checks.
- SP3: parent handoff only; owner decision required before writing a license.
- Forecast: approximately 100 authored lines; local-only, no delivery strategy beyond the existing feature branch.

## Verification / progress
- Base includes the verified Terraform scan change on `fxckcode/bounded-terraform-scan`, commit `a8fc26f5230b842da3e22f6a0db4994c47cc32e3`; branch was clean.
- The user corrected the prior omission and explicitly authorized completing the missing setup. An MIT recommendation was presented, but the clarification timed out without an answer; no license choice is assumed.
- Matt and Ponytail recommended the minimal README, Go 1.22 CI gates, and owner-selected license.
- CodeGraph is unavailable; no project-specific `.atl/skill-registry.md` exists. Engram mirror remains pending because it does not resolve Stackpeek.
- Parent read back the README and workflow; content matches the local entrypoint and detector scope. The license section explicitly says no license was selected.
- Offline `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...` passed for all three packages; `git diff --check` passed. YAML parsed locally with `CPAN::Meta::YAML`; `actionlint` and PyYAML are unavailable and were not installed.
- SP1/SP2 are complete. SP3 remains blocked until the owner selects license terms; do not infer MIT or copy Envy's license.
- No remote branch, push, PR, or release was created. Next: commit the README/CI setup locally and hand back the license decision.
