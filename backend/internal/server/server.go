package server

import (
	"log"

	"github.com/emersion/go-smtp"
	"github.com/habibiefaried/email-server/internal/storage"
)

func RunSMTPServer(fqdn string, port string, store storage.Storage) {
	be := &Backend{Store: store}
	s := smtp.NewServer(be)
	s.Addr = ":" + port
	s.AllowInsecureAuth = true

	if fqdn != "" {
		log.Printf("Starting SMTP server on %s\n", s.Addr)
		log.Printf("FQDN: %s\n", fqdn)
	} else {
		log.Printf("Starting SMTP server on %s (accepting all domains)\n", s.Addr)
	}

	if err := s.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start SMTP server: %v", err)
	}
}
