package jwt

// Config defines the inputs required to sign and validate JWTs.
type Config struct {
	Secret string `mapstructure:"secret"`
	// Expire is the token lifetime in hours. Zero means no automatic expiry.
	Expire int    `mapstructure:"expire"`
	Issuer string `mapstructure:"issuer"`
}
