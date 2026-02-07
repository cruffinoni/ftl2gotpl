# ftl2gotpl

Convert FreeMarker (`.ftl`) templates into Go `html/template` syntax.

## Current Scope
- Converts core directives: `if`/`elseif`/`else`, `list`, `assign`, `local`, `setting`.
- Converts interpolations: `${...}` / `#{...}`.
- Maps common built-ins used in this repo:
  - `?size`, `?has_content`, `?contains`, `?substring`, `?index_of`, `?index`, `?trim`
  - `?number`, `?number_to_datetime`, `?string`
  - `??`, `!default`, `?no_esc`
- Maps bracket access expressions to Go `index`, for example:
  - `user.metadata.attributes["userType"]`
  - `users[user_index]`
- Runs parse-check with `html/template`.
- Optional render-check using sidecar JSON data.
- Produces optional JSON and CSV reports.

## Known Limitations
- `<#function ...>` blocks are unsupported, except `formatPrice` which is replaced by a built-in helper stub.
- Expression-level function calls are limited to `formatPrice(...)`; other function calls are rejected.
- Macro calls (`<@...>`) are currently unsupported.
- Complex arithmetic expressions are intentionally restricted.
- `?index` is only supported on list loop item variables (e.g. inside `<#list items as item>`, `item?index`).

## Build
```bash
cd ftl2gotpl
make test
make build
```

## Usage
Use Make targets for common workflows:
```bash
cd ftl2gotpl
make help
make run
make convert
make render-check
```

Override input/output and strict mode when needed:
```bash
make convert IN=../templates_download OUT=./out-one STRICT=true
```

Useful Make variables:
- `IN` (default: `../templates_download`)
- `OUT` (default: `./out`)
- `EXT` (default: `.gotmpl`)
- `STRICT` (default: `false`)
- `SAMPLES` (default: `./testdata/samples`)
- `ARTIFACTS` (default: `./artifacts`)

Direct CLI usage (equivalent, with explicit flags):
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
make render-check SAMPLES=./testdata/samples
```

Sidecar lookup rule:
- For template `some/path/mail.ftl`, sample path is:
  - `<samples-root>/some/path/mail.ftl.json`
- When a sample exists and render succeeds, rendered HTML is written to:
  - `<out>/some/path/mail.rendered.html`

## Exit Codes
- `0`: success
- `1`: unexpected runtime/CLI error
- `2`: conversion failures
- `3`: parse-check or render-check failures

## Logs
- Log levels are colorized when output is an interactive terminal.
- Set `NO_COLOR=1` to disable colors.
- Set `CLICOLOR_FORCE=1` to force colors (even when not attached to a TTY).

## Reports
- JSON report contains:
  - file status
  - diagnostics (code/message/location)
  - detected features
  - required helper functions
  - render metadata (`render_checked`, `sample_path`, `rendered_path`)
- CSV report contains one row per file:
  - status
  - diagnostics count
  - helper count
  - features count
  - render-check metadata

Possible per-file status values:
- `converted`
- `converted_no_sample`
- `failed_conversion`
- `failed_parse`
- `failed_render`
