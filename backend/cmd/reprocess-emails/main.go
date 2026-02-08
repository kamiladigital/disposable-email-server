package main

import (
	"bytes"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jhillyerd/enmime"
	_ "github.com/lib/pq"
)

type emailRow struct {
	ID         string
	RawContent string
}

func main() {
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}
	log.Println("Connected to database")

	// Read email size limit from environment (default 512KB = 524288 bytes)
	maxEmailSize := int64(524288)
	if envSize := os.Getenv("EMAIL_SIZE_LIMIT"); envSize != "" {
		if parsed, err := strconv.ParseInt(envSize, 10, 64); err == nil {
			maxEmailSize = parsed
			log.Printf("Email size limit set to %d bytes", maxEmailSize)
		} else {
			log.Printf("Warning: Invalid EMAIL_SIZE_LIMIT value %q, using default 512KB", envSize)
		}
	}

	// Find emails with raw_content but empty/null body
	rows, err := db.Query(`
		SELECT id, raw_content
		FROM email
		WHERE raw_content IS NOT NULL
		  AND raw_content != ''
		  AND (body IS NULL OR body = '')
	`)
	if err != nil {
		log.Fatalf("Failed to query emails: %v", err)
	}
	defer rows.Close()

	var emails []emailRow
	for rows.Next() {
		var e emailRow
		if err := rows.Scan(&e.ID, &e.RawContent); err != nil {
			log.Fatalf("Failed to scan row: %v", err)
		}
		emails = append(emails, e)
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("Row iteration error: %v", err)
	}

	total := len(emails)
	if total == 0 {
		log.Println("No emails found with empty body. Nothing to do.")
		return
	}

	log.Printf("Found %d email(s) with raw_content but empty body. Processing...", total)

	updated := 0
	skipped := 0
	failed := 0

	for i, e := range emails {
		log.Printf("[%d/%d] Processing email %s...", i+1, total, e.ID)

		var body string

		// Check email size limit
		if int64(len(e.RawContent)) > maxEmailSize {
			log.Printf("  SKIP: Email exceeds size limit (%d > %d bytes)", len(e.RawContent), maxEmailSize)
			skipped++
			continue
		}

		// Parse with enmime
		env, err := enmime.ReadEnvelope(strings.NewReader(e.RawContent))
		if err != nil {
			log.Printf("  SKIP: Failed to parse raw_content: %v", err)
			skipped++
			continue
		}

		// Check for parsing errors
		if len(env.Errors) > 0 {
			log.Printf("  Warning: %d parsing errors", len(env.Errors))
		}

		// Convert to HTML with inline images
		body = emailToHTML(env)

		if body == "" {
			log.Printf("  SKIP: No body generated from email")
			skipped++
			continue
		}

		// Update the email body
		result, err := db.Exec(`
			UPDATE email SET body = $1 WHERE id = $2
		`, body, e.ID)
		if err != nil {
			log.Printf("  FAIL: Failed to update email: %v", err)
			failed++
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			log.Printf("  WARN: No rows affected for email %s", e.ID)
			failed++
			continue
		}

		log.Printf("  OK: Updated body (%d chars)", len(body))
		updated++
	}

	log.Println("========================================")
	log.Printf("Done. Total: %d | Updated: %d | Skipped: %d | Failed: %d", total, updated, skipped, failed)

	// Also show remaining emails that still have empty body
	var remaining int
	db.QueryRow(`
		SELECT COUNT(*) FROM email
		WHERE raw_content IS NOT NULL AND raw_content != ''
		  AND (body IS NULL OR body = '')
	`).Scan(&remaining)
	if remaining > 0 {
		log.Printf("WARNING: %d email(s) still have empty body after processing", remaining)
	} else {
		log.Println("All emails with raw_content now have body populated.")
	}
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
