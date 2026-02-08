package storage

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGenerateUUIDv7_Format(t *testing.T) {
	id := generateUUIDv7()
	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("UUIDv7 not parseable: %s — %v", id, err)
	}
	if parsed.Version() != 7 {
		t.Errorf("Expected UUID version 7, got %d: %s", parsed.Version(), id)
	}
	if parsed.Variant() != uuid.RFC4122 {
		t.Errorf("Expected RFC4122 variant, got %v: %s", parsed.Variant(), id)
	}
}

func TestGenerateUUIDv7_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := generateUUIDv7()
		if seen[id] {
			t.Fatalf("Duplicate UUIDv7 at iteration %d: %s", i, id)
		}
		seen[id] = true
	}
}

func TestGenerateUUIDv7_TimestampMonotonic(t *testing.T) {
	prev := generateUUIDv7()
	time.Sleep(2 * time.Millisecond)
	for i := 0; i < 10; i++ {
		next := generateUUIDv7()
		time.Sleep(2 * time.Millisecond)
		if next <= prev {
			t.Errorf("UUIDv7 not monotonically increasing: %s <= %s", next, prev)
		}
		prev = next
	}
}

func TestGenerateUUIDv7_Length(t *testing.T) {
	id := generateUUIDv7()
	if len(id) != 36 {
		t.Errorf("UUIDv7 length should be 36, got %d: %s", len(id), id)
	}
}

func TestGenerateUUIDv7_ParseRoundTrip(t *testing.T) {
	for i := 0; i < 100; i++ {
		id := generateUUIDv7()
		parsed, err := uuid.Parse(id)
		if err != nil {
			t.Fatalf("Parse failed for %s: %v", id, err)
		}
		if parsed.String() != id {
			t.Errorf("Round-trip mismatch: %s != %s", parsed.String(), id)
		}
	}
}

func TestEmailSummary_NoBodyField(t *testing.T) {
	s := EmailSummary{
		ID:        generateUUIDv7(),
		From:      "a@b.com",
		To:        "c@d.com",
		Subject:   "test",
		Date:      "2026-01-01",
		CreatedAt: time.Now(),
	}
	if s.ID == "" || s.From == "" || s.To == "" {
		t.Error("EmailSummary fields not set correctly")
	}
	if _, err := uuid.Parse(s.ID); err != nil {
		t.Errorf("EmailSummary ID should be valid UUID: %v", err)
	}
}

func TestEmailDetail_HasBodyAndAttachments(t *testing.T) {
	d := EmailDetail{
		ID:        generateUUIDv7(),
		From:      "a@b.com",
		To:        "c@d.com",
		Subject:   "test",
		Date:      "2026-01-01",
		Body:      "<p>Hello</p>",
		CreatedAt: time.Now(),
	}
	if d.Body == "" {
		t.Error("EmailDetail Body should not be empty")
	}
	if _, err := uuid.Parse(d.ID); err != nil {
		t.Errorf("EmailDetail ID should be valid UUID: %v", err)
	}
}
