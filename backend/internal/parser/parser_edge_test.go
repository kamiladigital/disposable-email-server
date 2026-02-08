package parser

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParse_Base64EncodedBody(t *testing.T) {
	// Simulate an email with base64-encoded body
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: Base64 Test\r\n" +
		"Date: Thu, 6 Feb 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte("Hello, this is base64 encoded!"))

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(parsed.Body, "Hello, this is base64 encoded!") {
		t.Errorf("Base64 body not decoded. Got: %q", parsed.Body)
	}
}

func TestParse_Base64EncodedHTMLBody(t *testing.T) {
	htmlContent := "<html><body><h1>Hello Base64</h1></body></html>"
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: Base64 HTML Test\r\n" +
		"Date: Thu, 6 Feb 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte(htmlContent))

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.HTMLBody != htmlContent {
		t.Errorf("Base64 HTML body not decoded. Got: %q, Want: %q", parsed.HTMLBody, htmlContent)
	}
}

func TestParse_MultipartWithBase64Parts(t *testing.T) {
	boundary := "boundary123"
	plainBody := "Plain text decoded from base64"
	htmlBody := "<p>HTML decoded from base64</p>"

	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: Multipart Base64\r\n" +
		"Date: Thu, 6 Feb 2026 10:00:00 +0000\r\n" +
		"Content-Type: multipart/alternative; boundary=\"" + boundary + "\"\r\n" +
		"\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte(plainBody)) + "\r\n" +
		"--" + boundary + "\r\n" +
		"Content-Type: text/html; charset=\"UTF-8\"\r\n" +
		"Content-Transfer-Encoding: base64\r\n" +
		"\r\n" +
		base64.StdEncoding.EncodeToString([]byte(htmlBody)) + "\r\n" +
		"--" + boundary + "--\r\n"

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Body != plainBody {
		t.Errorf("Plain text not decoded. Got: %q, Want: %q", parsed.Body, plainBody)
	}
	if parsed.HTMLBody != htmlBody {
		t.Errorf("HTML not decoded. Got: %q, Want: %q", parsed.HTMLBody, htmlBody)
	}
}

func TestParse_NonBase64Body(t *testing.T) {
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: Plain Test\r\n" +
		"Date: Thu, 6 Feb 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"\r\n" +
		"This is a plain text email, no base64."

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(parsed.Body, "This is a plain text email") {
		t.Errorf("Plain text body incorrect. Got: %q", parsed.Body)
	}
}

func TestParse_EmptyBody(t *testing.T) {
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: Empty Body\r\n" +
		"Date: Thu, 6 Feb 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"\r\n"

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Empty body should not cause errors
	if parsed.Subject != "Empty Body" {
		t.Errorf("Subject incorrect. Got: %q", parsed.Subject)
	}
}

func TestParse_MissingHeaders(t *testing.T) {
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"\r\n" +
		"Body with minimal headers"

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Subject != "" {
		t.Errorf("Subject should be empty for missing header. Got: %q", parsed.Subject)
	}
	if parsed.Date != "" {
		t.Errorf("Date should be empty for missing header. Got: %q", parsed.Date)
	}
}

func TestParse_QuotedPrintableBody(t *testing.T) {
	// Ensure non-base64 transfer encoding doesn't break
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: QP Test\r\n" +
		"Date: Thu, 6 Feb 2026 10:00:00 +0000\r\n" +
		"Content-Type: text/plain; charset=\"UTF-8\"\r\n" +
		"Content-Transfer-Encoding: quoted-printable\r\n" +
		"\r\n" +
		"Hello =C3=A9l=C3=A8ve"

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Body == "" {
		t.Error("Body should not be empty for quoted-printable content")
	}
	if !strings.Contains(parsed.Body, "élève") {
		t.Errorf("Body should contain decoded quoted-printable text, got: %q", parsed.Body)
	}
}

func TestParse_RawContentPreservedExact(t *testing.T) {
	rawContent := "From: sender@test.com\r\n" +
		"To: recipient@test.com\r\n" +
		"Subject: Preserve Raw\r\n" +
		"\r\n" +
		"Test body"

	parsed, err := Parse(rawContent)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.RawContent != rawContent {
		t.Error("Raw content not preserved exactly")
	}
}
