# warpin-database

GORM repositories, connections and transactions at the module root; query conditions under query.

Module path: `github.com/time-origin/warpin-go-common/warpin-database`.
Release tags: `warpin-database/vX.Y.Z`. The initial release version is `v0.1.0`.

For local development, run `go test ./...` here with the root workspace enabled.
For independent checks before publication, run from the repository root:

```bash
python3 scripts/check.py --module warpin-database test ./...
```

See the root README for migration, dependency injection and release instructions.
