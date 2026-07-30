# warpin-go-common

Reusable Go infrastructure packages for Warpin services.

This repository follows the capability-oriented organization of
`warpin-rs-common`, while remaining one idiomatic Go module so consumers can
pin and upgrade one version.

## Packages

- `auth`: JWT, cookie sessions, and OAuth provider primitives.
- `database`: reusable GORM connection, repository, query, and transaction helpers.
- `errors`: stable business-code error primitives.
- `http`: unified result and HTTP response helpers.
- `mail`: SMTP, AWS SES, and Aliyun DirectMail adapters.
- `storage`: object-storage adapters.
- `types`: shared persistence types.
- `utils`: focused utility packages migrated from existing Warpin services.

Application models, schema migrations, seed data, route policy, and other
service-specific behavior do not belong in this repository.

## Compatibility

The initial `v0.1.x` line is extracted from the existing VoiceCraft server.
It preserves its current JWT, result envelope, GORM repository, and utility
behavior unless a security or portability issue requires an explicit fix.

## Validation

```bash
go test ./...
go vet ./...
```
