# ftl2gotpl

Convert FreeMarker (`.ftl`) templates into Go `html/template` syntax.

## Current Scope
- Converts core directives: `if`/`elseif`/`else`, `list`, `assign`, `local`, `setting`.
- Converts interpolations: `${...}` / `#{...}`.
- Maps common built-ins used in this repo:
  - `?size`, `?has_content`, `?contains`, `?substring`, `?index_of`, `?trim`
  - `?number`, `?number_to_datetime`, `?string`
  - `??`, `!default`, `?no_esc`
- Runs parse-check with `html/template`.
- Optional render-check using sidecar JSON data.
- Produces optional JSON and CSV reports.

## Known Limitations
- `<#function ...>` blocks are currently unsupported and fail conversion (strict fail-fast behavior).
- Macro calls (`<@...>`) are currently unsupported.
- Complex arithmetic expressions are intentionally restricted.

## Build
```bash
cd ftl2gotpl
go test ./...
go build ./cmd/ftl2gotpl
```

## Usage
```bash
go run ./cmd/ftl2gotpl \
  --in ../templates_download \
  --out ./out \
  --ext .gotmpl \
  --strict=false \
  --report-json ./artifacts/report.json \
  --report-csv ./artifacts/report.csv
```

### Render Check
Enable render validation using sidecar JSON files:
```bash
go run ./cmd/ftl2gotpl \
  --in ../templates_download \
  --out ./out \
  --render-check \
  --samples-root ./testdata/samples
```

Sidecar lookup rule:
- For template `some/path/mail.ftl`, sample path is:
  - `<samples-root>/some/path/mail.ftl.json`

## Exit Codes
- `0`: success
- `2`: conversion failures
- `3`: parse-check or render-check failures

## Reports
- JSON report contains:
  - file status
  - diagnostics (code/message/location)
  - detected features
  - required helper functions
- CSV report contains one row per file:
  - status
  - diagnostics count
  - helper count
  - features count
  - render-check metadata
