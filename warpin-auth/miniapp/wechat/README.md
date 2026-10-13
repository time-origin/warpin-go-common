# WeChat Mini Program calls

Inject credentials from the consuming application's configuration. This package
does not read environment variables, files or configuration services. It owns no
business user, database, business application type or business JWT settings.

```go
client, err := wechat.New(wechat.Config{
    AppID: businessConfig.AppID,
    AppSecret: businessConfig.AppSecret,
}, httpClient)
// Handle err before use. Keep one client per app to reuse its token cache.
session, err := client.Code2Session(ctx, loginCode)
phone, err := client.GetPhoneNumber(ctx, phoneAuthorizationCode)
```

The login code comes from `wx.login`; the independent phone authorization code
comes from the user-triggered `getPhoneNumber` callback. The application decides
identity binding, registration, allowed app types and business token issuance.
`SessionKey` is excluded from JSON output and must never be sent to the frontend.

`GetAccessToken(ctx, forceRefresh)` uses the stable-token endpoint, caches by
client/AppID, and refreshes before expiry. `GetPhoneNumber` validates the returned
watermark AppID and retries once only for invalid/expired access tokens. Errors
expose numeric API error codes, not raw messages or credential-bearing URLs.
The HTTP client rejects redirects and bounds request duration and response size.

Candidate release: `warpin-auth/v0.1.1`; additions leave existing mobile App OAuth
contracts unchanged. Run `GOWORK=off go test -race ./miniapp/wechat` and the module's
ordinary test/vet/build checks before release.

References: [code2Session](https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-login/code2Session.html),
[getPhoneNumber](https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/user-info/phone-number/getPhoneNumber.html),
[stable token](https://developers.weixin.qq.com/miniprogram/dev/OpenApiDoc/mp-access-token/getStableAccessToken.html).
