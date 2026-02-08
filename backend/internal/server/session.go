package server

import (
	"io"
	"log"

	"github.com/emersion/go-smtp"
	"github.com/habibiefaried/email-server/internal/storage"
)

type Session struct {
	From  string
	To    string
	Store storage.Storage
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.From = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.To = to
	return nil
}

func (s *Session) Data(r io.Reader) error {
	body, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	email := storage.Email{
		From:    s.From,
		To:      s.To,
		Content: string(body),
	}
	filename, err := s.Store.Save(email)
	if err != nil {
		return err
	}
	log.Printf("from: %s, to: %s, saved in %s", s.From, s.To, filename)
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}
