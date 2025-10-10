package google

import (
	"notification-service/internal/template"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
}

func NewEmailService(email, password string) *EmailService {
	d := gomail.NewDialer("smtp.gmail.com", 587, email, password)
	return &EmailService{dialer: d}
}

func (e *EmailService) GreetingEmail(to, name string) error {
	m := gomail.NewMessage()
	m.SetHeader("From", e.dialer.Username)
	m.SetHeader("To", to)
	m.SetHeader("Subject", "Email xin chào")
	m.SetBody("text/html", template.GreetingTemplate(name))
	return e.dialer.DialAndSend(m)
}
