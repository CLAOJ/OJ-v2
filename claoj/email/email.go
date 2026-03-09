package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"time"

	"github.com/CLAOJ/claoj/config"
)

// EmailData holds common template data
type EmailData struct {
	SiteName  string
	SiteURL   string
	Username  string
	Email     string
	Subject   string
	Body      string
	Link      string
	LinkText  string
	ValidFor  string
	IPAddress string
	Year      int
}

// SendEmail sends an email using SMTP configuration
func SendEmail(to, subject, bodyHTML, bodyText string) error {
	cfg := config.C.Email

	if cfg.NoReply {
		log.Printf("email: no-reply mode, would send to %s subject: %s", to, subject)
		return nil
	}

	if cfg.SMTPHost == "" {
		return fmt.Errorf("email: SMTP host not configured")
	}

	from := fmt.Sprintf("%s <%s>", cfg.FromName, cfg.FromEmail)

	// Build email message
	var buf bytes.Buffer
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: multipart/alternative; boundary=\"CLAOJ_BOUNDARY\"\r\n")
	buf.WriteString("\r\n")

	// Plain text part
	buf.WriteString("--CLAOJ_BOUNDARY\r\n")
	buf.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(bodyText)
	buf.WriteString("\r\n\r\n")

	// HTML part
	buf.WriteString("--CLAOJ_BOUNDARY\r\n")
	buf.WriteString("Content-Type: text/html; charset=UTF-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n")
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString("\r\n")
	// Note: for production, should properly base64 encode HTML
	buf.WriteString(bodyHTML)
	buf.WriteString("\r\n\r\n")
	buf.WriteString("--CLAOJ_BOUNDARY--\r\n")

	addr := fmt.Sprintf("%s:%d", cfg.SMTPHost, cfg.SMTPPort)

	// Try TLS connection
	conn, err := tls.Dial("tcp", addr, &tls.Config{
		ServerName: cfg.SMTPHost,
	})
	if err != nil {
		return fmt.Errorf("email: failed to connect to SMTP: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		return fmt.Errorf("email: failed to create SMTP client: %w", err)
	}
	defer client.Close()

	// Auth if credentials provided
	if cfg.SMTPUser != "" {
		auth := smtp.PlainAuth("", cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPHost)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("email: failed to authenticate: %w", err)
		}
	}

	// Set sender and recipient
	if err := client.Mail(cfg.FromEmail); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}

	// Send email body
	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = buf.WriteTo(writer)
	return err
}

// SendVerificationEmail sends email verification email
func SendVerificationEmail(to, username, verifyLink string) error {
	data := EmailData{
		SiteName: config.C.Email.FromName,
		SiteURL:  config.C.App.SiteFullURL,
		Username: username,
		Link:     verifyLink,
		LinkText: "Verify Email",
		ValidFor: "24 hours",
		Year:     time.Now().Year(),
	}

	subject := "Verify your email - " + config.C.Email.FromName

	htmlBody, err := renderTemplate(verificationTemplate, data)
	if err != nil {
		return err
	}

	textBody := fmt.Sprintf("Hello %s,\n\nPlease verify your email by clicking: %s\n\nThis link expires in 24 hours.\n\nThanks,\n%s",
		username, verifyLink, config.C.Email.FromName)

	return SendEmail(to, subject, htmlBody, textBody)
}

// SendPasswordResetEmail sends password reset email
func SendPasswordResetEmail(to, username, resetLink string) error {
	data := EmailData{
		SiteName: config.C.Email.FromName,
		SiteURL:  config.C.App.SiteFullURL,
		Username: username,
		Link:     resetLink,
		LinkText: "Reset Password",
		ValidFor: "1 hour",
		Year:     time.Now().Year(),
	}

	subject := "Password reset request - " + config.C.Email.FromName

	htmlBody, err := renderTemplate(passwordResetTemplate, data)
	if err != nil {
		return err
	}

	textBody := fmt.Sprintf("Hello %s,\n\nYou requested a password reset. Click here to reset: %s\n\nThis link expires in 1 hour.\n\nIf you didn't request this, please ignore this email.\n\nThanks,\n%s",
		username, resetLink, config.C.Email.FromName)

	return SendEmail(to, subject, htmlBody, textBody)
}

// renderTemplate renders an HTML template with data
func renderTemplate(tmpl string, data EmailData) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Email templates
const verificationTemplate = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .header { background: #009688; color: white; padding: 20px; text-align: center; }
    .content { padding: 30px 20px; background: #f9f9f9; }
    .button { display: inline-block; padding: 12px 30px; background: #009688; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
    .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>{{.SiteName}}</h1>
    </div>
    <div class="content">
      <h2>Hello {{.Username}},</h2>
      <p>Thank you for registering! Please verify your email address by clicking the button below:</p>
      <p style="text-align: center;">
        <a href="{{.Link}}" class="button">{{.LinkText}}</a>
      </p>
      <p>This link will expire in {{.ValidFor}}.</p>
      <p>If you didn't create an account, you can safely ignore this email.</p>
    </div>
    <div class="footer">
      <p>&copy; {{.Year}} {{.SiteName}}. All rights reserved.</p>
      <p><a href="{{.SiteURL}}">{{.SiteURL}}</a></p>
    </div>
  </div>
</body>
</html>
`

const passwordResetTemplate = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .header { background: #009688; color: white; padding: 20px; text-align: center; }
    .content { padding: 30px 20px; background: #f9f9f9; }
    .button { display: inline-block; padding: 12px 30px; background: #009688; color: white; text-decoration: none; border-radius: 5px; margin: 20px 0; }
    .warning { background: #fff3cd; border: 1px solid #ffc107; padding: 15px; border-radius: 5px; margin: 20px 0; }
    .footer { text-align: center; padding: 20px; color: #666; font-size: 12px; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1>{{.SiteName}}</h1>
    </div>
    <div class="content">
      <h2>Hello {{.Username}},</h2>
      <p>You requested a password reset. Click the button below to reset your password:</p>
      <p style="text-align: center;">
        <a href="{{.Link}}" class="button">{{.LinkText}}</a>
      </p>
      <p>This link will expire in {{.ValidFor}}.</p>
      <div class="warning">
        <strong>Didn't request a password reset?</strong><br>
        You can safely ignore this email. Your password will remain unchanged.
      </div>
    </div>
    <div class="footer">
      <p>&copy; {{.Year}} {{.SiteName}}. All rights reserved.</p>
      <p><a href="{{.SiteURL}}">{{.SiteURL}}</a></p>
    </div>
  </div>
</body>
</html>
`
