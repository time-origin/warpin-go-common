package mail

import (
	"bytes"
	"fmt"
	"strings"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dm "github.com/alibabacloud-go/dm-20151123/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
)

type AliyunMailer struct {
	client       *dm.Client
	from         string
	fromName     string
	templatePath string
}

func NewAliyunMailer(cfg AliyunConfig) (*AliyunMailer, error) {
	if strings.TrimSpace(cfg.AccessKeyID) == "" ||
		strings.TrimSpace(cfg.AccessKeySecret) == "" {
		return nil, fmt.Errorf("aliyun access key is required")
	}
	if strings.TrimSpace(cfg.From) == "" {
		return nil, fmt.Errorf("aliyun sender address is required")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "dm.aliyuncs.com"
	}

	client, err := dm.NewClient(&openapi.Config{
		AccessKeyId:     tea.String(cfg.AccessKeyID),
		AccessKeySecret: tea.String(cfg.AccessKeySecret),
		Endpoint:        tea.String(endpoint),
	})
	if err != nil {
		return nil, fmt.Errorf("create aliyun direct-mail client: %w", err)
	}

	return &AliyunMailer{
		client:       client,
		from:         cfg.From,
		fromName:     cfg.FromName,
		templatePath: cfg.TemplatePath,
	}, nil
}

func (m *AliyunMailer) Send(
	to []string,
	subject string,
	templateName string,
	data map[string]interface{},
) error {
	if len(to) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	t, err := parseTemplate(m.templatePath, templateName)
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	data["Subject"] = subject

	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("execute email template: %w", err)
	}

	_, err = m.client.SingleSendMailWithOptions(
		&dm.SingleSendMailRequest{
			AccountName:    tea.String(m.from),
			AddressType:    tea.Int32(1),
			ReplyToAddress: tea.Bool(false),
			FromAlias:      tea.String(m.fromName),
			Subject:        tea.String(subject),
			HtmlBody:       tea.String(body.String()),
			ToAddress:      tea.String(strings.Join(to, ",")),
		},
		&util.RuntimeOptions{},
	)
	if err != nil {
		return fmt.Errorf("send email through aliyun direct mail: %w", err)
	}
	return nil
}

func (m *AliyunMailer) SendVerificationEmail(
	toEmail string,
	verificationCode string,
	expiresInMinutes int,
) error {
	return m.Send(
		[]string{toEmail},
		"Your Email Verification Code",
		"verification_code.html",
		map[string]interface{}{
			"VerificationCode": verificationCode,
			"ExpiresInMinutes": expiresInMinutes,
		},
	)
}

func (m *AliyunMailer) SendPasswordResetEmail(
	toEmail string,
	resetLink string,
	expiresInMinutes int,
) error {
	return m.Send(
		[]string{toEmail},
		"Reset Your Password",
		"password_reset_link.html",
		map[string]interface{}{
			"ResetLink":        resetLink,
			"ExpiresInMinutes": expiresInMinutes,
		},
	)
}
