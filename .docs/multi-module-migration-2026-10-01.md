# Multi-module migration plan

Approved by the user on 2026-10-01. Branch: codex/common-worktree-consolidation. Migrate every existing capability and test into the ten agreed modules.

## Mapping

| Original | Destination |
|---|---|
| errors | warpin-errors |
| types | warpin-types |
| utils/* | warpin-utils/* |
| auth/* | warpin-auth/* |
| request/session | warpin-auth/request/session |
| database/* | warpin-database/* |
| query | warpin-database/query |
| http/result, http/response | warpin-http/result, warpin-http/response |
| http/hertz/response | warpin-http-hertz/response |
| payment/ysepay | warpin-payment/ysepay |
| mail/* including embedded templates | warpin-mail/* |
| storage/gcs | warpin-object-storage/gcs |

## Constraints

Move existing implementations and tests; change imports only. Preserve the H5 payment changes. Each module owns go.mod/go.sum and LICENSE; root go.work supports joint development. Only HTTP depends on errors; Hertz depends on HTTP and errors. All other modules are independent. Do not add local replace directives to published go.mod files. Unpublished errors/HTTP v0.1.0 graph edges use version-specific replacements only in go.work.

Internal module dependencies target the provisional v0.1.0 version, with release tags <directory>/v0.1.0. This request does not authorize publishing, pushing, upgrading the business server, or deleting old worktrees and unrelated documents. Existing consumers pinned to v0.6.0 continue using that release; new consumers must migrate imports and requirements.

## Verification

Compare all original source/resources with their mapped destinations, allowing only import substitutions. Tidy each module while retaining pinned external versions. Update README and CI. Run all workspace tests and independently test each module with GOWORK=off through a temporary file proxy and temporary source copies. This checks unpublished module archives without persisting replace directives or synthetic checksums. Run payment/HTTP race tests and all module vet checks. Verify payment has no dependency on Hertz, mail or object storage. Record the approved architecture decision in server memory without changing the server dependency.

## Completed verification

- All 75 original Go source/test files and mail templates match the baseline 6a10f2b after only intended import substitutions. All ten modules include the original license.
- All-module workspace tests passed using explicit warpin-*/... package patterns.
- `python3 scripts/check.py test -count=1 ./...`: all ten modules passed independently with GOWORK=off; independently tidied go.mod files matched the source manifests.
- `python3 scripts/check.py vet ./...`: passed for all ten modules.
- `python3 scripts/check.py --module warpin-payment --module warpin-http --module warpin-http-hertz test -race -count=1 ./...`: passed.
- No old import paths or persisted go.mod replace directives remain.
- Payment module graph includes only the module itself, PKCS12 and golang.org/x dependencies; no Hertz, mail, GCS or sibling common modules.
- Existing exported APIs and behavior are unchanged. Packages that previously had no tests still have no tests; this migration does not establish live database, OAuth, mail, storage or payment acceptance.
- CI configuration is updated; remote CI results must be checked separately after pushing the feature branch. No module tags are published and the business server remains pinned to root v0.6.0. The user authorized submitting the migration to the feature branch after successful revalidation on 2026-10-01.
