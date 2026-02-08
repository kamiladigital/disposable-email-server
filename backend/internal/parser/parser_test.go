package parser

import (
	"os"
	"strings"
	"testing"
)

func TestParse_Gmail_SimpleEmail(t *testing.T) {
	content, err := os.ReadFile("../../samples/gmail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := Parse(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(parsed.From, "habibie.faried@paidy.com") {
		t.Errorf("From field incorrect. Got: %s", parsed.From)
	}
	if !strings.Contains(parsed.To, "admin@test.milahabibie.com") {
		t.Errorf("To field incorrect. Got: %s", parsed.To)
	}
	if parsed.Subject != "test" {
		t.Errorf("Subject incorrect. Got: %s", parsed.Subject)
	}
	if parsed.Date == "" {
		t.Error("Date field is empty")
	}
	if !strings.Contains(parsed.Body, "test") {
		t.Errorf("Body does not contain expected text. Got: %s", parsed.Body)
	}
	if len(parsed.Attachments) != 0 {
		t.Errorf("Expected no attachments, but got %d", len(parsed.Attachments))
	}
	if parsed.RawContent == "" {
		t.Error("Raw content is empty")
	}

	t.Log("✓ Gmail simple email parsed correctly")
}

func TestParse_AnonymousMail_NoAttachments(t *testing.T) {
	content, err := os.ReadFile("../../samples/anonymousemail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := Parse(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(parsed.From, "noreply@anonymousemail.se") {
		t.Errorf("From field incorrect. Got: %s", parsed.From)
	}
	if !strings.Contains(parsed.To, "admin2@test.milahabibie.com") {
		t.Errorf("To field incorrect. Got: %s", parsed.To)
	}
	if parsed.Subject != "test" {
		t.Errorf("Subject incorrect. Got: %s", parsed.Subject)
	}
	if !strings.Contains(parsed.Body, "test") {
		t.Errorf("Body does not contain expected text. Got: %s", parsed.Body)
	}
	if len(parsed.Attachments) != 0 {
		t.Errorf("Expected no attachments, but got %d", len(parsed.Attachments))
	}

	t.Log("✓ AnonymousMail email parsed correctly")
}

func TestParse_WithAttachments(t *testing.T) {
	content, err := os.ReadFile("../../samples/attachments.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := Parse(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(parsed.From, "habibie.faried@paidy.com") {
		t.Errorf("From field incorrect. Got: %s", parsed.From)
	}
	if !strings.Contains(parsed.To, "admin@test.milahabibie.com") {
		t.Errorf("To field incorrect. Got: %s", parsed.To)
	}
	if parsed.Subject != "Re: test" {
		t.Errorf("Subject incorrect. Got: %s", parsed.Subject)
	}

	if len(parsed.Attachments) == 0 {
		t.Error("Expected attachments, but got none")
	}

	t.Logf("✓ Email with attachments parsed correctly - found %d attachments", len(parsed.Attachments))

	foundPNG := false
	for _, att := range parsed.Attachments {
		if strings.Contains(att.Filename, "Screenshot") && strings.HasSuffix(att.Filename, ".png") {
			foundPNG = true
			if att.ContentType != "image/png" {
				t.Errorf("Attachment content type incorrect. Got: %s", att.ContentType)
			}
			if len(att.Data) == 0 {
				t.Error("Attachment data is empty")
			}
			if len(att.Data) >= 4 {
				expected := []byte{0x89, 0x50, 0x4E, 0x47}
				actual := att.Data[:4]
				if string(actual) != string(expected) {
					t.Errorf("PNG file signature incorrect. Got: %v, Expected: %v", actual, expected)
				}
			}
			t.Logf("✓ PNG attachment verified - Filename: %s, Size: %d bytes", att.Filename, len(att.Data))
		}
	}

	if !foundPNG {
		t.Error("PNG attachment not found")
	}
}

func TestParse_AttachmentDecoding(t *testing.T) {
	content, err := os.ReadFile("../../samples/attachments.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := Parse(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Attachments) == 0 {
		t.Fatal("No attachments found")
	}

	for i, att := range parsed.Attachments {
		if len(att.Data) == 0 {
			t.Errorf("Attachment %d has no data", i)
		}
		if att.ContentType == "image/png" && len(att.Data) > 4 {
			if att.Data[0] != 0x89 || att.Data[1] != 0x50 {
				t.Errorf("Attachment %d: Data not properly decoded from base64", i)
			}
		}
	}

	t.Log("✓ All attachments properly decoded from base64")
}

func TestParse_HeaderPreservation(t *testing.T) {
	content, err := os.ReadFile("../../samples/gmail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := Parse(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.From == "" {
		t.Error("From header is empty")
	}
	if parsed.To == "" {
		t.Error("To header is empty")
	}
	if parsed.Subject == "" {
		t.Error("Subject header is empty")
	}
	if parsed.Date == "" {
		t.Error("Date header is empty")
	}

	t.Logf("✓ Headers preserved - From: %s, To: %s, Subject: %s, Date: %s",
		parsed.From, parsed.To, parsed.Subject, parsed.Date)
}

func TestParse_MultipleAttachmentTypes(t *testing.T) {
	content, err := os.ReadFile("../../samples/attachments.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	parsed, err := Parse(string(content))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	uniqueFilenames := make(map[string]bool)
	for _, att := range parsed.Attachments {
		uniqueFilenames[att.Filename] = true
	}

	t.Logf("✓ Found %d total attachments with %d unique filenames", len(parsed.Attachments), len(uniqueFilenames))

	for i, att := range parsed.Attachments {
		if att.Filename == "" {
			t.Errorf("Attachment %d has no filename", i)
		}
		if att.ContentType == "" {
			t.Errorf("Attachment %d has no content type", i)
		}
		if len(att.Data) == 0 {
			t.Errorf("Attachment %d has no data", i)
		}
	}
}

func TestParse_RawContentPreserved(t *testing.T) {
	content, err := os.ReadFile("../../samples/gmail.txt")
	if err != nil {
		t.Fatalf("Failed to read sample file: %v", err)
	}

	contentStr := string(content)
	parsed, err := Parse(contentStr)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.RawContent != contentStr {
		t.Error("Raw content not preserved exactly")
	}

	t.Log("✓ Raw content preserved exactly")
}

func BenchmarkParse_SimpleEmail(b *testing.B) {
	content, _ := os.ReadFile("../../samples/gmail.txt")
	contentStr := string(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Parse(contentStr)
	}
}

func BenchmarkParse_WithAttachments(b *testing.B) {
	content, _ := os.ReadFile("../../samples/attachments.txt")
	contentStr := string(content)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Parse(contentStr)
	}
}
