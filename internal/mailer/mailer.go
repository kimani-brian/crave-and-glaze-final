package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"path/filepath"
)

type Mailer struct {
	Host     string
	Port     string
	Username string
	Password string
}

func New(host, port, user, pass string) *Mailer {
	return &Mailer{
		Host:     host,
		Port:     port,
		Username: user,
		Password: pass,
	}
}

// Send sends an HTML email to a specific recipient
func (m *Mailer) Send(to, subject, templateFile string, data interface{}) error {
	// 1. Parse the HTML template for the email body
	tmplPath := filepath.Join("web", "templates", "email", templateFile)
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return err
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return err
	}

	// 2. Setup Authentication
	auth := smtp.PlainAuth("", m.Username, m.Password, m.Host)

	// 3. Construct the Message Headers
	headers := "MIME-Version: 1.0\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		fmt.Sprintf("From: Crave & Glaze <%s>\r\n", m.Username) +
		fmt.Sprintf("To: %s\r\n", to) +
		fmt.Sprintf("Subject: %s\r\n", subject) +
		"\r\n"

	msg := []byte(headers + body.String())

	// 4. Send
	addr := fmt.Sprintf("%s:%s", m.Host, m.Port)
	err = smtp.SendMail(addr, auth, m.Username, []string{to}, msg)
	return err
}
