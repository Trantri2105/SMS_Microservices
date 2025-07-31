package mail

import (
	"gopkg.in/mail.v2"
	"io"
)

type Attachment struct {
	Name    string
	Content io.Reader
}

type Sender interface {
	SendMail(to []string, subject, htmlBody, textBody string, attachments []Attachment) error
}

type Dialer interface {
	DialAndSend(m ...*mail.Message) error
}

type sender struct {
	email  string
	dialer Dialer
}

func (s *sender) SendMail(to []string, subject, htmlBody, textBody string, attachments []Attachment) error {
	m := mail.NewMessage()

	m.SetHeader("From", s.email)
	m.SetHeader("To", to...)
	m.SetHeader("Subject", subject)

	if textBody != "" {
		m.AddAlternative("text/plain", textBody)
	}
	if htmlBody != "" {
		m.SetBody("text/html", htmlBody)
	}

	for _, attachment := range attachments {
		if attachment.Content != nil && attachment.Name != "" {
			m.Attach(attachment.Name, mail.SetCopyFunc(func(w io.Writer) error {
				_, err := io.Copy(w, attachment.Content)
				return err
			}))
		}
	}

	if err := s.dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

func NewMailSender(email, password, host string, port int) Sender {
	return &sender{
		email:  email,
		dialer: mail.NewDialer(host, port, email, password),
	}
}
