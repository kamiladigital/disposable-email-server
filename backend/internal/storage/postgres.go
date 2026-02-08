package storage

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"net/mail"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jhillyerd/enmime"
	_ "github.com/lib/pq"
)

// PostgresStorage implements Storage interface with Postgres backend
type PostgresStorage struct {
	db           *sql.DB
	maxEmailSize int64 // Maximum email size in bytes (default 512KB)
}

// NewPostgresStorage creates a new postgres storage instance
// dsn format: "user=username password=pass dbname=emaildb host=localhost port=5432 sslmode=disable"
// EMAIL_SIZE_LIMIT env var controls max email size in bytes (default 524288 = 512KB)
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, err
	}

	// Read email size limit from environment (default 512KB = 524288 bytes)
	maxEmailSize := int64(524288) // 512KB default
	if envSize := os.Getenv("EMAIL_SIZE_LIMIT"); envSize != "" {
		if parsed, err := strconv.ParseInt(envSize, 10, 64); err == nil {
			maxEmailSize = parsed
			log.Printf("Email size limit set to %d bytes", maxEmailSize)
		} else {
			log.Printf("Warning: Invalid EMAIL_SIZE_LIMIT value %q, using default 512KB", envSize)
		}
	}

	ps := &PostgresStorage{
		db:           db,
		maxEmailSize: maxEmailSize,
	}
	if err := ps.createTables(); err != nil {
		return nil, err
	}

	log.Printf("Connected to Postgres database (max email size: %d bytes)", ps.maxEmailSize)
	return ps, nil
}

// createTables creates the email table if it doesn't exist
func (ps *PostgresStorage) createTables() error {
	emailTableSQL := `
	CREATE TABLE IF NOT EXISTS email (
		id UUID PRIMARY KEY,
		"from" TEXT NOT NULL,
		"to" TEXT NOT NULL,
		subject TEXT,
		date TEXT,
		body TEXT,
		raw_content TEXT,
		headers TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := ps.db.Exec(emailTableSQL); err != nil {
		return err
	}

	indexSQL := `
	CREATE INDEX IF NOT EXISTS idx_email_from ON email("from");
	CREATE INDEX IF NOT EXISTS idx_email_to ON email("to");
	CREATE INDEX IF NOT EXISTS idx_email_created_at ON email(created_at);
	CREATE INDEX IF NOT EXISTS idx_email_id ON email(id);`

	if _, err := ps.db.Exec(indexSQL); err != nil {
		return err
	}

	return nil
}

// generateUUIDv7 generates a UUIDv7 using github.com/google/uuid
func generateUUIDv7() string {
	id, err := uuid.NewV7()
	if err != nil {
		// Fallback to V4 if V7 fails (should never happen)
		return uuid.New().String()
	}
	return id.String()
}

func extractHeadersFromRawContent(raw string) (string, string, string, string) {
	if raw == "" {
		return "", "", "", ""
	}

	// Read only the header section to avoid expensive parsing.
	headerEnd := strings.Index(raw, "\r\n\r\n")
	if headerEnd == -1 {
		headerEnd = strings.Index(raw, "\n\n")
	}
	if headerEnd == -1 {
		headerEnd = len(raw)
	}

	headers := raw[:headerEnd] + "\r\n\r\n"
	msg, err := mail.ReadMessage(strings.NewReader(headers))
	if err != nil {
		return "", "", "", ""
	}

	return msg.Header.Get("From"), msg.Header.Get("To"), msg.Header.Get("Subject"), msg.Header.Get("Date")
}

// Save saves an email and its attachments to postgres
// Base64 content is decoded by the parser BEFORE inserting into the database
func (ps *PostgresStorage) Save(email Email) (string, error) {
	// Check email size limit before parsing
	if ps.maxEmailSize > 0 && int64(len(email.Content)) > ps.maxEmailSize {
		log.Printf("Warning: Email exceeds size limit (%d > %d bytes), storing error message", len(email.Content), ps.maxEmailSize)
		from, to, subject, date := extractHeadersFromRawContent(email.Content)
		if from == "" {
			from = email.From
		}
		if to == "" {
			to = email.To
		}
		emailID := generateUUIDv7()
		_, err := ps.db.Exec(
			`INSERT INTO email (id, "from", "to", subject, date, body, raw_content)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			emailID, from, to, subject, date, "Sorry, the email exceeds our limit (512kb)", "",
		)
		if err != nil {
			return "", err
		}
		return emailID, nil
	}

	// Parse email using enmime
	env, err := enmime.ReadEnvelope(strings.NewReader(email.Content))
	if err != nil {
		log.Printf("Warning: enmime parse error: %v, storing raw content", err)
		emailID := generateUUIDv7()
		_, err := ps.db.Exec(
			`INSERT INTO email (id, "from", "to", subject, date, body, raw_content)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			emailID, email.From, email.To, "", "", "Email parsing failed", email.Content,
		)
		if err != nil {
			return "", err
		}
		return emailID, nil
	}

	// Check for parsing errors (enmime continues even with errors)
	if len(env.Errors) > 0 {
		log.Printf("Warning: %d enmime parsing errors encountered", len(env.Errors))
		for _, e := range env.Errors {
			log.Printf("  - %s", e.String())
		}
	}

	// Convert to HTML with inline images
	htmlBody := emailToHTML(env)

	// Extract metadata
	from := env.GetHeader("From")
	to := env.GetHeader("To")
	subject := env.GetHeader("Subject")
	date := env.GetHeader("Date")

	if from == "" {
		from = email.From
	}
	if to == "" {
		to = email.To
	}

	emailID := generateUUIDv7()
	_, err = ps.db.Exec(
		`INSERT INTO email (id, "from", "to", subject, date, body, raw_content)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		emailID, from, to, subject, date, htmlBody, email.Content,
	)
	if err != nil {
		return "", err
	}

	log.Printf("Email saved to postgres: id=%s, from=%s, to=%s, inline_images=%d",
		emailID, from, to, len(env.Inlines))

	return emailID, nil
}

// EmailSummary represents email metadata for inbox listing (no body/attachments)
type EmailSummary struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Date      string    `json:"date"`
	CreatedAt time.Time `json:"created_at"`
}

// EmailDetail represents a full email with body and attachment metadata
type EmailDetail struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	Date      string    `json:"date"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

// GetInbox fetches email summaries for a recipient (5 per page).
// The page parameter is 1-based: page=1 returns rows 1-5, page=2 returns rows 6-10, etc.
func (ps *PostgresStorage) GetInbox(address string, page int) ([]EmailSummary, error) {
	if page < 1 {
		page = 1
	}
	const pageSize = 5
	sqlOffset := (page - 1) * pageSize
	rows, err := ps.db.Query(`
		SELECT id, "from", "to", COALESCE(subject, ''), COALESCE(date, ''), created_at
		FROM email
		WHERE "to" = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, address, pageSize, sqlOffset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	emails := make([]EmailSummary, 0)
	for rows.Next() {
		var email EmailSummary
		if err := rows.Scan(&email.ID, &email.From, &email.To, &email.Subject, &email.Date, &email.CreatedAt); err != nil {
			return nil, err
		}
		emails = append(emails, email)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return emails, nil
}

// GetEmailByID fetches full email detail including body and attachments by UUIDv7
func (ps *PostgresStorage) GetEmailByID(id string) (*EmailDetail, error) {
	var email EmailDetail
	var rawContent sql.NullString
	err := ps.db.QueryRow(`
		SELECT id, "from", "to", COALESCE(subject, ''), COALESCE(date, ''),
		       COALESCE(body, ''), raw_content, created_at
		FROM email WHERE id = $1
	`, id).Scan(
		&email.ID, &email.From, &email.To, &email.Subject, &email.Date,
		&email.Body, &rawContent, &email.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	// If body is empty, try to reprocess raw_content
	if email.Body == "" && rawContent.Valid && rawContent.String != "" {
		rawContentStr := rawContent.String

		// Re-parse with enmime
		if env, err := enmime.ReadEnvelope(strings.NewReader(rawContentStr)); err == nil {
			email.Body = emailToHTML(env)
			// Update database so we don't reprocess again
			ps.db.Exec(`UPDATE email SET body = $1 WHERE id = $2`, email.Body, id)
		} else {
			log.Printf("Failed to reprocess email %s: %v", id, err)
			email.Body = "<pre>Email parsing failed</pre>"
		}
	}

	return &email, nil
}

// Close closes the database connection
func (ps *PostgresStorage) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}
	return nil
}

// emailToHTML converts an enmime envelope to HTML with inline images embedded as data URIs
func emailToHTML(env *enmime.Envelope) string {
	// Get HTML content (enmime will auto-convert plain text to HTML if needed)
	htmlContent := env.HTML
	if htmlContent == "" {
		htmlContent = "<pre>" + env.Text + "</pre>"
	}

	// Build a map of Content-ID to image data
	images := make(map[string][]byte)
	imageTypes := make(map[string]string)

	// Process inline images
	for _, inline := range env.Inlines {
		// Get Content-ID from the part
		if cid := inline.ContentID; cid != "" {
			images[cid] = inline.Content
			imageTypes[cid] = inline.ContentType
		}
	}

	// Replace cid: references with data URLs
	htmlContent = replaceCIDWithDataURL(htmlContent, images, imageTypes)

	return htmlContent
}

// replaceCIDWithDataURL replaces cid: references in HTML with base64 data URIs
func replaceCIDWithDataURL(html string, images map[string][]byte, imageTypes map[string]string) string {
	// Find all cid: references
	re := regexp.MustCompile(`cid:([^"'\s>]+)`)

	result := re.ReplaceAllStringFunc(html, func(match string) string {
		// Extract the CID (remove "cid:" prefix)
		cid := match[4:]

		// Look up the image data
		if imageData, ok := images[cid]; ok {
			mimeType := imageTypes[cid]
			if mimeType == "" {
				mimeType = detectImageType(imageData)
			}

			// Convert to base64 data URL
			encoded := base64.StdEncoding.EncodeToString(imageData)
			return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
		}

		return match
	})

	return result
}

// detectImageType detects image MIME type from binary data
func detectImageType(data []byte) string {
	if len(data) < 4 {
		return "image/jpeg"
	}

	// Check for common image signatures
	if bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return "image/jpeg"
	}
	if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}) {
		return "image/png"
	}
	if bytes.HasPrefix(data, []byte{0x47, 0x49, 0x46}) {
		return "image/gif"
	}
	if bytes.HasPrefix(data, []byte{0x42, 0x4D}) {
		return "image/bmp"
	}
	if bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 12 && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return "image/jpeg" // default
}
