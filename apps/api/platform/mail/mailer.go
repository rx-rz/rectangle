package mail

import (
	"bytes"
	"context"
	_ "embed"
	"html/template"
	"rx-rz/rectangle-api/internal/config"

	"github.com/resend/resend-go/v3"
)

//go:embed templates/otp.html
var otpTemplate string

type OTPEmailParams struct {
	Digits       OTPDigits
	Device       string
	RequestedAt  string
	IPAddress    string
	Region       string
	DashboardURL string
	DocsURL      string
	SupportURL   string
}

type OTPDigits struct {
	D1 string
	D2 string
	D3 string
	D4 string
	D5 string
	D6 string
}

func (m *Mailer) SplitOTP(code string) OTPDigits {
	if len(code) != 6 {
		return OTPDigits{}
	}
	return OTPDigits{
		D1: string(code[0]),
		D2: string(code[1]),
		D3: string(code[2]),
		D4: string(code[3]),
		D5: string(code[4]),
		D6: string(code[5]),
	}
}

type Mailer struct {
	Client *resend.Client
	Cfg    *config.Config
}

func NewMailer(cfg config.Config) *Mailer {
	return &Mailer{
		Client: resend.NewClient(cfg.MailerApiKey),
		Cfg:    &cfg,
	}
}

func (m *Mailer) SendOTPMail(ctx context.Context, data OTPEmailParams, to string) error {
	html, err := RenderOTPEmail(data)
	if err != nil {
		return err
	}

	params := &resend.SendEmailRequest{
		From:    m.Cfg.MailerFrom,
		To:      []string{to},
		Subject: "Your Rectangle verification code",
		Html:    html,
	}

	_, err = m.Client.Emails.SendWithContext(ctx, params)
	return err
}

func RenderOTPEmail(data OTPEmailParams) (string, error) {
	tmpl, err := template.New("otp.html").Parse(otpTemplate)
	if err != nil {
		return "", err
	}
	var body bytes.Buffer
	err = tmpl.Execute(&body, data)
	if err != nil {
		return "", err
	}
	return body.String(), nil
}
