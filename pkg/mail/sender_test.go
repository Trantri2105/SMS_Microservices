package mail

import (
	"bytes"
	"errors"
	"github.com/stretchr/testify/assert"
	"gopkg.in/mail.v2"
	"strings"
	"testing"
)

type mockDialer struct {
	SentMessage *mail.Message
	ShouldError bool
}

func (d *mockDialer) DialAndSend(m ...*mail.Message) error {
	if d.ShouldError {
		return errors.New("error")
	}
	if len(m) > 0 {
		d.SentMessage = m[0]
	}
	return nil
}

func TestSendMail(t *testing.T) {
	t.Run("sends an email successfully", func(t *testing.T) {
		mock := &mockDialer{}
		s := &sender{
			email:  "from@example.com",
			dialer: mock,
		}

		to := []string{"to@example.com"}
		subject := "Test Subject"
		htmlBody := "<h1>Hello</h1>"
		textBody := "Hello"
		attachmentContent := "this is a test file"
		attachments := []Attachment{
			{
				Name:    "test.txt",
				Content: strings.NewReader(attachmentContent),
			},
		}
		err := s.SendMail(to, subject, htmlBody, textBody, attachments)
		assert.NoError(t, err)
		assert.NotNil(t, mock.SentMessage)
		assert.Equal(t, s.email, mock.SentMessage.GetHeader("From")[0])
		assert.Equal(t, to[0], mock.SentMessage.GetHeader("To")[0])
		assert.Equal(t, subject, mock.SentMessage.GetHeader("Subject")[0])

		var body bytes.Buffer
		mock.SentMessage.WriteTo(&body)
		assert.Contains(t, body.String(), "Content-Type: text/html")
		assert.Contains(t, body.String(), "<h1>Hello</h1>")
		assert.Contains(t, body.String(), "Content-Disposition: attachment; filename=\"test.txt\"")
	})

	t.Run("returns an error when dialer fails", func(t *testing.T) {
		mock := &mockDialer{ShouldError: true}
		s := &sender{
			email:  "from@example.com",
			dialer: mock,
		}
		err := s.SendMail([]string{"to@example.com"}, "Subject", "Body", "", nil)
		assert.Error(t, err)
	})
}
