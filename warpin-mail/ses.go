package mail

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
)

// SESMailer 使用 AWS SES 发送邮件。
type SESMailer struct {
	client       *sesv2.Client
	fromName     string // 新增
	fromAddress  string // 重命名 from
	templatePath string
}

// NewSESMailer 创建一个新的 SESMailer 实例。
func NewSESMailer(cfg SESConfig) (*SESMailer, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config for SES: %w", err)
	}

	return &SESMailer{
		client:       sesv2.NewFromConfig(awsCfg),
		fromName:     cfg.FromName,
		fromAddress:  cfg.From,
		templatePath: cfg.TemplatePath,
	}, nil
}

// Send 使用 SES 发送邮件。
func (m *SESMailer) Send(to []string, subject, templateName string, data map[string]interface{}) error {
	t, err := parseTemplate(m.templatePath, templateName)
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := t.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// 构建 "From" 字段，格式为 "FromName <email@address.com>"
	fromEmailAddress := fmt.Sprintf("\"%s\" <%s>", m.fromName, m.fromAddress)

	input := &sesv2.SendEmailInput{
		FromEmailAddress: aws.String(fromEmailAddress), // 使用新的格式
		Destination: &types.Destination{
			ToAddresses: to,
		},
		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(subject),
					Charset: aws.String("UTF-8"),
				},
				Body: &types.Body{
					Html: &types.Content{
						Data:    aws.String(body.String()),
						Charset: aws.String("UTF-8"),
					},
				},
			},
		},
	}

	_, err = m.client.SendEmail(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	return nil
}

// ... (SendVerificationEmail 和 SendPasswordResetEmail 保持不变) ...
func (m *SESMailer) SendVerificationEmail(toEmail, verificationCode string, expiresInMinutes int) error {
	subject := "Your Email Verification Code"
	templateName := "verification_code.html"
	data := map[string]interface{}{
		"VerificationCode": verificationCode,
		"ExpiresInMinutes": expiresInMinutes,
	}
	return m.Send([]string{toEmail}, subject, templateName, data)
}
func (m *SESMailer) SendPasswordResetEmail(toEmail, resetLink string, expiresInMinutes int) error {
	subject := "Reset Your Password"
	templateName := "password_reset_link.html"
	data := map[string]interface{}{
		"ResetLink":        resetLink,
		"ExpiresInMinutes": expiresInMinutes,
	}
	return m.Send([]string{toEmail}, subject, templateName, data)
}
