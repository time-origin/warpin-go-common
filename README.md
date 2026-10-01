# warpin-go-common

Reusable infrastructure for Warpin services, organized as one repository with
ten independently consumable Go modules, following the capability boundaries of
`warpin-rs-common`. Every previous package, test and embedded mail template is
migrated; implementations retain their existing behavior.

## Modules

Each directory owns its `go.mod`, dependency graph and release tag.

| Module | Packages / capabilities | Internal dependencies |
|---|---|---|
| `warpin-errors` | Shared codes and error primitives | None |
| `warpin-types` | JSONB persistence type | None |
| `warpin-utils` | account, converter, excel, fileutil, oauthstate, sanitizer, stringutil | None |
| `warpin-auth` | jwt, session, request/session, oauth plus Douyin, GitHub, Google, WeChat and Xiaohongshu | None |
| `warpin-database` | GORM connections, repositories, transactions, validation, query conditions | None |
| `warpin-http` | result, standard net/http response helpers | warpin-errors |
| `warpin-http-hertz` | Native Hertz response adapter | warpin-http, warpin-errors |
| `warpin-payment` | ysepay: pre-order, notification verification, trade/refund queries, refunds and optional H5 routing | None |
| `warpin-mail` | SMTP, AWS SES, Aliyun DirectMail and embedded templates | None |
| `warpin-object-storage` | gcs: storage, upload and signed URLs | None |

Application models, schema migrations, seed data and route policy remain in
business services. Do not introduce reverse dependencies from common modules
to a service, or circular module dependencies.

## Consumption and releases

The initial release uses `v0.1.0` for each module. A consumer can install
only the selected module and its transitive dependencies:

```bash
go get github.com/time-origin/warpin-go-common/warpin-payment@v0.1.0
```

```go
import "github.com/time-origin/warpin-go-common/warpin-payment/ysepay"
```

A consumer of payment does not depend on Hertz, mail or object storage.
Packages inside one module still share its go.mod dependency graph: for example,
warpin-utils currently groups Excel and archive helpers with other utilities.

Version tags must include the module directory: `warpin-payment/v0.1.0`,
`warpin-errors/v0.1.0`, and so on. Internal HTTP dependencies use `v0.1.0`. The initial release publishes all
ten matching module tags together. Future releases must provide compatible
errors and HTTP versions before their dependents. Every module ships its own
LICENSE. Existing root-module release tags remain unchanged.

The old root module is removed on this branch. Previously published root-module
versions, including v0.6.0, remain usable. To upgrade, change imports using the
mapping in [.docs/multi-module-migration-2026-10-01.md](.docs/multi-module-migration-2026-10-01.md),
add the selected module requirements, then run go mod tidy and business tests.
No duplicate compatibility implementations are maintained.

## Local development

Root `go.work` lists all ten modules and resolves their local sources without
adding replace directives to production go.mod files. Version-specific replacements for errors and HTTP live only in go.work
to resolve their v0.1.0 graph edges locally during joint development. Business consumers use
their own go.mod and do not need this workspace.

```bash
# Run one module in the workspace.
cd warpin-payment
go test ./...
```

From the repository root, `go test ./...` no longer selects the child modules;
use explicit module patterns or the independent check script below. Go workspaces
do not centralize dependency versions like Cargo workspace.dependencies: pinned
dependencies live in each module's go.mod.

## HTTP transport boundaries

Core packages, including `warpin-errors`, `warpin-auth`, and `warpin-http/result`, must not import a
web framework. Standard HTTP helpers belong in `warpin-http/response`; Hertz-specific
code belongs under `warpin-http-hertz`. Business domain packages do not belong in this
module.

Existing services can keep using the compatible `warpin-http/response` API without
`go-chi/render`. New Hertz services should use `warpin-http-hertz` directly instead of
routing their high-frequency paths through a `net/http` adaptor. Shared
middleware logic should remain framework-independent, with transport-specific
wrappers added only when a concrete caller needs them.

Services that own a different public response contract should use
`warpin-http-hertz/response.JSON` with their own value and status, or `NoContent` for
an empty 204 response. The `Success` and `Error` helpers retain the legacy
`code/data/count/msg` envelope and HTTP 200 behavior for existing consumers.

With Go 1.27, consumers can build with `-tags=stdjson,gjson` to select Hertz's
portable JSON implementations; the pinned Sonic version otherwise reports a
compatibility warning and falls back to standard JSON encoding.

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

## Ysepay dependency injection

`warpin-payment/ysepay` implements cashier pre-order, signed payment notification
parsing, trade query, refund acceptance, and refund query. It deliberately
does not implement merchant onboarding, order persistence, callback routes,
idempotency, or mobile SDK invocation.

The consuming service must load the initiator's PFX, its password, the Ysepay
public certificate, and merchant identities from its Secret system. The PFX
belongs to the initiator (`certId`); each payment request separately supplies
the contracted Ysepay secondary-merchant payee (`mercId`) and its approved
business code.

```go
signingPFX, err := os.ReadFile(appConfig.Ysepay.SigningPFXSecretPath)
if err != nil {
	return err
}
yseCertificate, err := os.ReadFile(appConfig.Ysepay.PublicCertificatePath)
if err != nil {
	return err
}

client, err := ysepay.NewClient(ysepay.Config{
	Environment:                ysepay.EnvironmentProduction,
	InitiatorMerchantID:        appConfig.Ysepay.InitiatorMerchantID,
	SigningPKCS12:              signingPFX,
	SigningCertificatePassword: secretStore.YsepayPFXPassword(),
	YsePublicCertificate:       yseCertificate,
})
if err != nil {
	return err
}

order, err := client.CreateCashierOrder(ctx, ysepay.CreateCashierOrderRequest{
	OrderID:             businessOrder.PaymentNumber,
	PayeeMerchantID:     terminalMerchant.YsepayMerchantID,
	BusinessCode:        ysepay.BusinessCodeStandard,
	ShopDate:            businessOrder.ShopDate,
	AmountFen:           businessOrder.AmountFen,
	PaymentValidMinutes: 30,
	BackURL:             appConfig.PublicBaseURL + "/payment/ysepay/notify",
	PayMode:             ysepay.PaymentModeAlipay,
	// H5Join and AppType are supplied by Ysepay for the merchant's enabled
	// H5 cashier scene. Leave them empty for the existing native flow.
	H5Join:              appConfig.Ysepay.H5Join,
	AppType:             appConfig.Ysepay.AppType,
})
```

Use `PaymentModeAlipay` (`26`) for the aggregated cashier Alipay WebView flow and
`PaymentModeWeChatMiniProgram` (`29`) for the WeChat cashier Mini Program.
The create-order response may contain `PayURL`, `EncryptedData`, or both;
for WeChat, pass the verified `BusinessData` to the mobile integration as
described by Ysepay. `CashierAppID` is returned when present but is not required.
Only a verified server notification or a verified query result can finalize a
payment. A mobile return URL is never proof of payment.

The consuming business is already contracted as a Ysepay secondary merchant.
Underlying WeChat and Alipay transaction merchant numbers belong to Ysepay;
the business does not apply for, report, or configure its own channel merchant
numbers in this client or its server environment.

Never commit PFX files, certificate passwords, real merchant numbers, AppIDs,
callback URLs, signatures, or encrypted gateway payloads to this repository.
The package accepts certificate bytes only and does not read fixed paths,
environment variables, or configuration services.

## Compatibility

This migration changes module and import paths, not exported APIs, JWT behavior,
result envelopes, GORM repository behavior, OAuth contracts or payment semantics.

## Validation

```bash
# All modules, workspace disabled; checks each go.mod is independently tidy.
python3 scripts/check.py test ./...
python3 scripts/check.py vet ./...

# Focused race checks.
python3 scripts/check.py --module warpin-payment --module warpin-http --module warpin-http-hertz test -race ./...
```

The script packages local modules into a temporary file proxy, checks temporary
source copies with GOWORK=off, and removes its proxy, module cache and synthetic
checksums afterwards. External dependencies are reused from the existing Go
download cache where available. No replacement or unpublished-module checksum
is written into the source modules. This validates independent dependency resolution from the current sources;
it does not verify remote publication or a live payment gateway. Remote release
verification is recorded in [.docs/module-release-v0.1.0-2026-10-01.md](.docs/module-release-v0.1.0-2026-10-01.md).
