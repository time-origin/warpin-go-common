# Initial multi-module release v0.1.0

User authorization: publish the new modules on 2026-10-01.
Source branch: codex/common-worktree-consolidation.

All ten modules use v0.1.0 and tags <module-directory>/v0.1.0. Existing root tags, including v0.6.0, must remain unchanged. All tags are pushed atomically; no merge into main or business dependency upgrade is required.

## Modules

- `github.com/time-origin/warpin-go-common/warpin-auth@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-database@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-errors@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-http@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-http-hertz@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-mail@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-object-storage@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-payment@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-types@v0.1.0`
- `github.com/time-origin/warpin-go-common/warpin-utils@v0.1.0`

## Validation

The source refactor commit 845aa29 passed local uncached workspace and independent module tests, all-module vet, payment/HTTP/Hertz race checks, and GitHub CI run 36845178955. The release preparation changes only documentation; all Go source files, manifests and verification scripts are unchanged from that tested commit.

After pushing the tags, verify their peeled commits and confirm every prior remote tag is preserved. Use clean temporary consumers with GOWORK=off and no local replace to download all ten modules and compile all their public packages; also consume the old root v0.6.0 in the same project. Verify a separate payment-only consumer excludes Hertz, mail, object storage and sibling modules. Archive the exact verification results below after completion. Live gateway, database, mail, object-storage and OAuth acceptance are outside package-release verification.

## Publication results

- Release commit: f51d010615a44177f19f12914abe9799440964d3.
- All ten annotated module tags were pushed atomically with the feature branch. Every tag resolves to the release commit.
- All previous remote tags, including root v0.6.0, were verified byte-for-byte unchanged.
- Release-commit GitHub CI passed: https://github.com/time-origin/warpin-go-common/actions/runs/36845916212.
- No merge into main and no business dependency upgrade were performed.

## Remote consumption verification

Clean temporary consumers used GOWORK=off, a fresh module cache, and no local replace directives. All ten actual v0.1.0 modules were downloaded from remote Git tags. All 26 new public packages compiled together with representative old-root v0.6.0 packages (payment, HTTP, database, OAuth, errors). `go mod verify` passed for the resulting dependency graph.

A separate payment-only consumer downloaded warpin-payment v0.1.0 and compiled successfully. Its graph contains no sibling common modules, Hertz, mail SDKs or Google object storage.

The initial public-proxy attempt timed out on this host. Successful verification used temporary per-process GONOPROXY/GONOSUMDB for github.com/time-origin/warpin-go-common*, fetching the repository over SSH; third-party artifacts were reused from the ordinary Go download cache where available. No global Go/Git configuration was changed. Remote Git publication and archive integrity are verified; public-proxy indexing and checksum-database reachability are not asserted.

Verified payment-only graph:

```text
example.com/warpin-payment-check
github.com/time-origin/warpin-go-common/warpin-payment v0.1.0
golang.org/x/crypto v0.43.0
golang.org/x/net v0.45.0
golang.org/x/sys v0.37.0
golang.org/x/term v0.36.0
golang.org/x/text v0.30.0
software.sslmate.com/src/go-pkcs12 v0.7.3
```
