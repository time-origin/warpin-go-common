# warpin-go-common

Reusable Go infrastructure packages for Warpin services.

This repository follows the capability-oriented organization of
`warpin-rs-common`, while remaining one idiomatic Go module so consumers can
pin and upgrade one version.

## Packages

- `auth`: JWT, cookie sessions, OAuth provider primitives, and WeChat,
  Douyin, and Xiaohongshu identity providers.
- `database`: reusable GORM connection, repository, query, and transaction helpers.
- `errors`: stable business-code error primitives.
- `http`: unified result and HTTP response helpers.
- `mail`: SMTP, AWS SES, and Aliyun DirectMail adapters.
- `storage`: object-storage adapters.
- `types`: shared persistence types.
- `utils`: focused utility packages migrated from existing Warpin services.

Application models, schema migrations, seed data, route policy, and other
service-specific behavior do not belong in this repository.

## OAuth dependency injection

OAuth providers do not read environment variables or configuration files.
The consuming service injects its runtime configuration into each provider,
registers the enabled providers, and passes only the short-lived authorization
credential when authenticating:

```go
wechatProvider, err := wechat.New(wechat.Config{
	AppID:     businessConfig.WeChat.AppID,
	AppSecret: businessConfig.WeChat.AppSecret,
}, httpClient)
if err != nil {
	return err
}

registry, err := oauth.NewRegistry(wechatProvider)
if err != nil {
	return err
}

identity, err := registry.Authenticate(ctx, "wechat", oauth.Credential{
	Code: authorizationCode,
})
```

The service owns provider enablement, secret storage, account binding,
persistence, and business token issuance. The common module returns only a
normalized third-party identity.

## Compatibility

The initial `v0.1.x` line is extracted from the existing VoiceCraft server.
It preserves its current JWT, result envelope, GORM repository, and utility
behavior unless a security or portability issue requires an explicit fix.

## Validation

```bash
go test ./...
go vet ./...
```
