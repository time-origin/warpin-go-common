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
