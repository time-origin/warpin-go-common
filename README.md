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
- `http/result`: framework-independent result and error mapping primitives.
- `http/response`: standard `net/http` response helpers.
- `http/hertz`: native CloudWeGo Hertz transport adapters.
- `mail`: SMTP, AWS SES, and Aliyun DirectMail adapters.
- `payment/ysepay`: Ysepay Xiao-Y aggregated cashier client for Alipay and
  WeChat Mini Program cashier APP flows.
- `storage`: object-storage adapters.
- `types`: shared persistence types.
- `utils`: focused utility packages migrated from existing Warpin services.

Application models, schema migrations, seed data, route policy, and other
service-specific behavior do not belong in this repository.

## HTTP transport boundaries

Core packages, including `errors`, `auth`, and `http/result`, must not import a
web framework. Standard HTTP helpers belong in `http/response`; Hertz-specific
code belongs under `http/hertz`. Business domain packages do not belong in this
module.

Existing services can keep using the compatible `http/response` API without
`go-chi/render`. New Hertz services should use `http/hertz` directly instead of
routing their high-frequency paths through a `net/http` adaptor. Shared
middleware logic should remain framework-independent, with transport-specific
wrappers added only when a concrete caller needs them.

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

`payment/ysepay` implements cashier pre-order, signed payment notification
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

The initial `v0.1.x` line is extracted from the existing VoiceCraft server.
It preserves its current JWT, result envelope, GORM repository, and utility
behavior unless a security or portability issue requires an explicit fix.

## Validation

```bash
go test ./...
go vet ./...
```
