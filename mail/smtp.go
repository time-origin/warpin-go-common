package mail

import (
	"bytes"
	"fmt"

	"gopkg.in/gomail.v2"
)

// MailerInterface 定义了所有邮件发送器都必须实现的方法。
type MailerInterface interface {
	Send(to []string, subject, templateName string, data map[string]interface{}) error
	SendVerificationEmail(toEmail, verificationCode string, expiresInMinutes int) error
	SendPasswordResetEmail(toEmail, resetLink string, expiresInMinutes int) error
}

// SMTPMailer 使用传统的 SMTP 服务器发送邮件。
type SMTPMailer struct {
	FromName     string // 新增
	Host         string
	Port         int
	Username     string
	Password     string
	FromAddress  string // 重命名 from
	SSL          bool
	TemplatePath string
}

// NewSMTPMailer 创建一个新的 SMTPMailer 实例。
func NewSMTPMailer(cfg SMTPConfig) *SMTPMailer {
	return &SMTPMailer{
		FromName:     cfg.FromName,
		Host:         cfg.Host,
		Port:         cfg.Port,
		Username:     cfg.Username,
		Password:     cfg.Password,
		FromAddress:  cfg.From,
		SSL:          cfg.SSL,
		TemplatePath: cfg.TemplatePath,
	}
}

// Send 使用 SMTP 发送邮件。
func (m *SMTPMailer) Send(to []string, subject, templateName string, data map[string]interface{}) error {
	t, err := parseTemplate(m.TemplatePath, templateName)
	if err != nil {
		return fmt.Errorf("parse email template: %w", err)
	}

	var body bytes.Buffer
	if data == nil {
		data = make(map[string]interface{})
	}
	data["Subject"] = subject
	err = t.Execute(&body, data)
	if err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	msg := gomail.NewMessage()
	// 使用 SetAddressHeader 来设置带名称的 From 地址
	msg.SetAddressHeader("From", m.FromAddress, m.FromName)
	msg.SetHeader("To", to...)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", body.String())

	d := gomail.NewDialer(m.Host, m.Port, m.Username, m.Password)
	d.SSL = m.SSL

	if err := d.DialAndSend(msg); err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// ... (SendVerificationEmail 和 SendPasswordResetEmail 保持不变) ...
func (m *SMTPMailer) SendVerificationEmail(toEmail, verificationCode string, expiresInMinutes int) error {
	subject := "Your Email Verification Code"
	templateName := "verification_code.html"
	data := map[string]interface{}{
		"VerificationCode": verificationCode,
		"ExpiresInMinutes": expiresInMinutes,
	}
	return m.Send([]string{toEmail}, subject, templateName, data)
}
func (m *SMTPMailer) SendPasswordResetEmail(toEmail, resetLink string, expiresInMinutes int) error {
	subject := "Reset Your Password"
	templateName := "password_reset_link.html"
	data := map[string]interface{}{
		"ResetLink":        resetLink,
		"ExpiresInMinutes": expiresInMinutes,
	}
	return m.Send([]string{toEmail}, subject, templateName, data)
}
