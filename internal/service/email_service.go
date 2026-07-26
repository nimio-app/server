package service

import (
	"fmt"

	"github.com/resend/resend-go/v2"
)

// EmailService handles email sending
type EmailService interface {
	SendVerificationEmail(email, username, token string) error
}

type emailService struct {
	resendClient *resend.Client
	fromEmail    string
	fromName     string
	appURL       string
}

// NewEmailService creates a new email service
func NewEmailService(apiKey, fromEmail, fromName, appURL string) EmailService {
	client := resend.NewClient(apiKey)
	return &emailService{
		resendClient: client,
		fromEmail:    fromEmail,
		fromName:     fromName,
		appURL:       appURL,
	}
}

// SendVerificationEmail sends an email verification link
func (s *emailService) SendVerificationEmail(email, username, token string) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.appURL, token)

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif; line-height: 1.6; color: #333; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); padding: 40px 20px; text-align: center; border-radius: 10px 10px 0 0;">
        <h1 style="color: white; margin: 0; font-size: 28px;">Welcome to Nimio! 🎉</h1>
    </div>
    
    <div style="background: #ffffff; padding: 40px 30px; border: 1px solid #e0e0e0; border-top: none; border-radius: 0 0 10px 10px;">
        <p style="font-size: 16px; margin-bottom: 20px;">Hi <strong>%s</strong>,</p>
        
        <p style="font-size: 16px; margin-bottom: 20px;">
            Thanks for joining Nimio—where respect starts before the first message.
        </p>
        
        <p style="font-size: 16px; margin-bottom: 30px;">
            Please verify your email address to complete your registration and start sharing your intentional availability with trusted connections.
        </p>
        
        <div style="text-align: center; margin: 40px 0;">
            <a href="%s" style="background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 14px 40px; text-decoration: none; border-radius: 6px; font-weight: 600; font-size: 16px; display: inline-block; box-shadow: 0 4px 6px rgba(0,0,0,0.1);">
                Verify Email Address
            </a>
        </div>
        
        <p style="font-size: 14px; color: #666; margin-top: 30px; padding-top: 20px; border-top: 1px solid #e0e0e0;">
            If the button doesn't work, copy and paste this link into your browser:
        </p>
        <p style="font-size: 13px; color: #667eea; word-break: break-all; background: #f5f5f5; padding: 10px; border-radius: 4px;">
            %s
        </p>
        
        <p style="font-size: 14px; color: #999; margin-top: 30px;">
            This link will expire in <strong>24 hours</strong>.
        </p>
        
        <p style="font-size: 14px; color: #666; margin-top: 20px;">
            If you didn't create an account with Nimio, you can safely ignore this email.
        </p>
    </div>
    
    <div style="text-align: center; margin-top: 30px; padding: 20px; color: #999; font-size: 12px;">
        <p style="margin: 5px 0;">
            <strong>Nimio</strong> - Intentional Availability
        </p>
        <p style="margin: 5px 0;">
            Presence is a gift, not an obligation.
        </p>
    </div>
</body>
</html>
`, username, verificationURL, verificationURL)

	textContent := fmt.Sprintf(`
Welcome to Nimio!

Hi %s,

Thanks for joining Nimio—where respect starts before the first message.

Please verify your email address by clicking the link below:
%s

This link will expire in 24 hours.

If you didn't create an account with Nimio, you can safely ignore this email.

---
Nimio - Intentional Availability
Presence is a gift, not an obligation.
`, username, verificationURL)

	params := &resend.SendEmailRequest{
		From:    fmt.Sprintf("%s <%s>", s.fromName, s.fromEmail),
		To:      []string{email},
		Subject: "Verify your Nimio email address",
		Html:    htmlContent,
		Text:    textContent,
	}

	_, err := s.resendClient.Emails.Send(params)
	if err != nil {
		return fmt.Errorf("failed to send verification email: %w", err)
	}

	return nil
}
