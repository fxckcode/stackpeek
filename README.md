# Stackpeek

Stackpeek scans a local repository for technology indicators and shows findings and diagnostics in a terminal UI. It accepts the current directory by default or one directory path.

## Install and run

Use Go 1.22 or newer. From a local Stackpeek checkout:

```sh
go install ./cmd/stackpeek
stackpeek [path]
```

Omit `path` to scan the current directory. To build a local executable instead:

```sh
go build -o stackpeek ./cmd/stackpeek
./stackpeek [path]
```

Press `q`, `Q`, or Ctrl-C to exit the UI.

## What it detects

Detections are based on these files and directory entries; they indicate signals, not a complete inventory of a repository's technologies.

| Technology | Signal |
| --- | --- |
| Node.js | `package.json` |
| NestJS | `@nestjs/core` or `@nestjs/common` in `package.json` dependencies or devDependencies |
| PHP | `composer.json` |
| Laravel | `laravel/framework` in `composer.json`, or an `artisan` file |
| Python | `requirements.txt` |
| Django | A Django package entry in `requirements.txt` |
| Docker | `Dockerfile` |
| Docker Compose | `docker-compose.yml` |
| Environment Template | `.env.example` |
| Prisma | `prisma/schema.prisma` |
| GitHub Actions | A `.yml` or `.yaml` file directly in `.github/workflows/` |
| Terraform | A `.tf` file found while traversing the repository |

Terraform traversal skips `node_modules`, `.git`, and `vendor` directories. It is capped at 50,000 entries and 64 path levels. If either limit is hit, Stackpeek reports an incomplete-scan diagnostic; findings from that traversal are not an exhaustive clean result. Diagnostics are displayed alongside findings.

## Development

The module targets Go 1.22. Run the project checks from the repository root:

```sh
gofmt -w $(git ls-files '*.go')
go vet ./...
go test ./...
go build ./...
```

## License

No license has been selected. Do not assume permission to use, modify, or redistribute this project.
