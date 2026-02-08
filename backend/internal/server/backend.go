package server

import (
	"github.com/emersion/go-smtp"
	"github.com/habibiefaried/email-server/internal/storage"
)

type Backend struct {
	Store storage.Storage
}

func (bkd *Backend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	return &Session{Store: bkd.Store}, nil
}
