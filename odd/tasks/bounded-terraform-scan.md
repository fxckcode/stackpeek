# Stackpeek bounded Terraform scanning

## Objective
Bound recursive Terraform detection and make incomplete scans observable instead of silently treating walk failures as no findings.

## Problem and rationale
The built-in registry describes filesystem reads as bounded, and `ReadFileBounded` caps individual files at 1 MiB. The Terraform detector still walks the entire root without a global entry/depth bound and discards `fs.WalkDir` errors (`internal/detector/builtin.go:107-131`). A large tree can consume unbounded traversal time; an unreadable tree can silently produce incomplete findings. `fs.WalkDir` is used and does not recurse through symlinked directories by default.

## Scope authorized
- `internal/detector/builtin.go`
- `internal/detector/builtin_test.go`
- `odd/tasks/bounded-terraform-scan.md`
- No README/CI/license additions in this work unit. The repo has no tracked README, CI, or LICENSE; a license remains an owner decision.
- No remote API, dependency download, push, PR, or release.

## Tasks
- [x] S1 — Add conservative entry/depth limits and diagnostics; cover exact boundaries, walk errors, and cancellation during traversal.

## Acceptance criteria
- Terraform traversal has explicit default limits of at most 50,000 entries and 64 path levels, with no external dependencies.
- Reaching a traversal limit stops scanning and emits an explicit diagnostic indicating results are incomplete.
- `fs.WalkDir` errors are not discarded; unreadable entries produce diagnostics, and cancellation does not masquerade as a clean scan.
- Existing ignored directories remain skipped, and symlinked directories are not followed.
- Tests prove below/at/over entry and depth limits, report a walk error while retaining a readable sibling, and cancel after traversal has started. Run only offline with `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off`; never permit Go to download modules or toolchains.

## Routing and risk
- S1: delegated direct to one writer because it changes detector behavior and tests; high verification tier because partial-scan semantics are user-visible. Require writer self-verification plus an independent verifier, then parent spot check.
- Forecast: approximately 80 authored lines; delivery strategy `single-pr` only if later explicitly requested. No push or PR is authorized now.

## Verification / progress
- Base: `03a7862937ea75cc9601688f7fbe8bf143cf8e81`; clean `main` at clone time.
- CodeGraph was unavailable (no upstream executable); bounded local source inspection used instead.
- Matt and Ponytail recommended bounding traversal and surfacing incomplete scans through the existing diagnostics API. Ponytail confirmed `fs.WalkDir` does not recurse through directory symlinks; that behavior remains.
- Go 1.22.2 matches `go.mod`; all tests ran offline with `GOPROXY=off` and `GOSUMDB=off`.
- Writer tests covered over-limit entry/depth paths, walk errors, and cancellation. The first independent verifier requested exact boundary and in-flight cancellation tests; the single correction added these.
- Final independent verifier verdict: PASS. It confirmed below/at/over limits, in-flight cancellation, walk-error sibling retention, ignored-directory pruning, and incomplete diagnostics. Commands passed: `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./internal/detector`; `GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test ./...` (all three packages); `gofmt -d internal/detector/builtin.go internal/detector/builtin_test.go`; `git diff --check`. Parent also reran the full offline suite successfully.
- Engram mirror pending because the memory store does not resolve this repository; do not guess a project key.
- S1 work-unit commit: `a8fc26f5230b842da3e22f6a0db4994c47cc32e3` (`fix: bound Terraform scans and report incomplete results`).
- No remote branch, push, or PR was created; explicit authorization is still required.
