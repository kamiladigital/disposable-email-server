package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"

	"github.com/habibiefaried/email-server/internal/dnsutil"
	"github.com/habibiefaried/email-server/internal/server"
	"github.com/habibiefaried/email-server/internal/storage"
)

const expectedDomainIP = "149.28.152.71"

func extractDomain(address string) (string, error) {
	parsed, err := mail.ParseAddress(address)
	if err == nil {
		address = parsed.Address
	}
	parts := strings.Split(address, "@")
	if len(parts) != 2 || parts[1] == "" {
		return "", fmt.Errorf("invalid email address")
	}
	return parts[1], nil
}

func main() {
	fqdn := ""

	// Initialize storage backend
	var store storage.Storage
	var pgStore *storage.PostgresStorage
	dbURL := os.Getenv("DB_URL")
	if dbURL != "" {
		var err error
		pgStore, err = storage.NewPostgresStorage(dbURL)
		if err != nil {
			log.Printf("Warning: Failed to connect to postgres: %v", err)
			log.Printf("Falling back to file-only storage")
			store = storage.NewFileStorage("emails")
		} else {
			log.Printf("PostgreSQL storage initialized (database-only mode)")
			store = pgStore
		}
	} else {
		log.Printf("DB_URL not set, using file-only storage")
		store = storage.NewFileStorage("emails")
	}

	// Get SMTP port from environment variable, default to 2525
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "2525"
	}

	// Always run the email server
	go server.RunSMTPServer(fqdn, smtpPort, store)

	// HTTP API setup
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "48080"
	}
	addr := ":" + port

	// Health check endpoint
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "OK")
	})

	// Inbox API endpoint (summary list, 5 per page)
	http.HandleFunc("/inbox", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		// Check if postgres is available
		if pgStore == nil {
			http.Error(w, "Postgres storage not configured", http.StatusServiceUnavailable)
			return
		}

		// Get email address from query parameter
		address := r.URL.Query().Get("email")
		if address == "" {
			http.Error(w, "Missing 'email' query parameter", http.StatusBadRequest)
			return
		}

		// Get page parameter (default 1)
		page := 1
		if pageStr := r.URL.Query().Get("page"); pageStr != "" {
			if parsedPage, err := strconv.Atoi(pageStr); err == nil && parsedPage >= 1 {
				page = parsedPage
			}
		}

		// Fetch email summaries (5 per page, no body/attachments)
		emails, err := pgStore.GetInbox(address, page)
		if err != nil {
			log.Printf("Error fetching inbox for %s: %v", address, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(emails); err != nil {
			log.Printf("Error encoding JSON: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Email detail endpoint (full content by UUIDv7)
	http.HandleFunc("/email", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		if pgStore == nil {
			http.Error(w, "Postgres storage not configured", http.StatusServiceUnavailable)
			return
		}

		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "Missing 'id' query parameter", http.StatusBadRequest)
			return
		}

		email, err := pgStore.GetEmailByID(id)
		if err != nil {
			log.Printf("Error fetching email %s: %v", id, err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if email == nil {
			http.Error(w, "Email not found", http.StatusNotFound)
			return
		}

		if err := json.NewEncoder(w).Encode(email); err != nil {
			log.Printf("Error encoding JSON: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	})

	// Domain validation endpoint
	http.HandleFunc("/domain/validate", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		emailAddress := r.URL.Query().Get("email")
		if emailAddress == "" {
			http.Error(w, "Missing 'email' query parameter", http.StatusBadRequest)
			return
		}

		domain, err := extractDomain(emailAddress)
		if err != nil {
			http.Error(w, "Invalid email address", http.StatusBadRequest)
			return
		}

		if err := dnsutil.ValidateFQDN(domain); err != nil {
			http.Error(w, "Invalid email domain", http.StatusBadRequest)
			return
		}

		// Check MX records and verify they resolve to expected IP
		// This supports both:
		// 1. domain.com → MX → domain.com → A → IP (direct)
		// 2. domain.com → MX → mx1.domain.com → A → IP (standard)
		mxOk, mxStatus, _ := dnsutil.CheckMXRecordWithIP(domain, expectedDomainIP)

		if mxOk {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status":    "ok",
				"domain":    domain,
				"mx_status": mxStatus,
			})
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":    "error",
			"domain":    domain,
			"message":   "MX records do not resolve to expected IP (149.28.152.71)",
			"mx_status": mxStatus,
		})
	})

	log.Printf("Starting HTTP API on %s", addr)
	log.Printf("Endpoints: / (health), /inbox?email=<address> (list), /email?id=<uuid> (detail), /domain/validate?email=<address>")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
