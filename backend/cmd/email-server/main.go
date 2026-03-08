package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

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

	// Image proxy endpoint – fetches external images server-side so the browser
	// never has to make a cross-origin request (avoids CORP/COEP blocking).
	http.HandleFunc("/proxy", func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Cross-Origin-Resource-Policy", "cross-origin")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "Missing 'url' query parameter", http.StatusBadRequest)
			return
		}

		parsed, err := url.Parse(rawURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			http.Error(w, "Invalid URL: only http and https are supported", http.StatusBadRequest)
			return
		}

		// Block requests to loopback / private RFC-1918 / link-local addresses (SSRF guard).
		// isPrivateHost resolves the hostname to IPs and returns true if any is non-public.
		isPrivateHost := func(hostname string) bool {
			ips, err := net.LookupHost(hostname)
			if err != nil {
				return false
			}
			for _, ipStr := range ips {
				ip := net.ParseIP(ipStr)
				if ip == nil {
					continue
				}
				if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() {
					return true
				}
			}
			return false
		}

		host := parsed.Hostname()
		if isPrivateHost(host) {
			http.Error(w, "Forbidden: private address", http.StatusForbidden)
			return
		}

		client := &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
					return fmt.Errorf("redirect to non-http scheme blocked")
				}
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				// Validate redirect destination against private IPs (SSRF via redirect guard)
				if isPrivateHost(req.URL.Hostname()) {
					return fmt.Errorf("redirect to private address blocked")
				}
				return nil
			},
		}

		req, err := http.NewRequest(http.MethodGet, rawURL, nil)
		if err != nil {
			http.Error(w, "Failed to create request", http.StatusInternalServerError)
			return
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; email-proxy/1.0)")

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Proxy fetch error for %s: %v", rawURL, err)
			http.Error(w, "Failed to fetch resource", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			http.Error(w, "Remote resource is not an image", http.StatusForbidden)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.WriteHeader(http.StatusOK)
		if _, err := io.Copy(w, resp.Body); err != nil {
			log.Printf("Proxy copy error for %s: %v", rawURL, err)
		}
	})

	log.Printf("Starting HTTP API on %s", addr)
	log.Printf("Endpoints: / (health), /inbox?email=<address> (list), /email?id=<uuid> (detail), /domain/validate?email=<address>")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Failed to start HTTP server: %v", err)
	}
}
