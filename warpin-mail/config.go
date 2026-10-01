package mail

type SMTPConfig struct {
	FromName     string `mapstructure:"from_name"`
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	From         string `mapstructure:"from"`
	SSL          bool   `mapstructure:"ssl"`
	TemplatePath string `mapstructure:"template_path"`
}

type SESConfig struct {
	FromName        string `mapstructure:"from_name"`
	Region          string `mapstructure:"region"`
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	From            string `mapstructure:"from"`
	TemplatePath    string `mapstructure:"template_path"`
}

type AliyunConfig struct {
	AccessKeyID     string `mapstructure:"access_key_id"`
	AccessKeySecret string `mapstructure:"access_key_secret"`
	Endpoint        string `mapstructure:"endpoint"`
	From            string `mapstructure:"from"`
	FromName        string `mapstructure:"from_name"`
	TemplatePath    string `mapstructure:"template_path"`
}

type Config struct {
	Provider string       `mapstructure:"provider"`
	SMTP     SMTPConfig   `mapstructure:"smtp"`
	SES      SESConfig    `mapstructure:"ses"`
	Aliyun   AliyunConfig `mapstructure:"aliyun"`
}
