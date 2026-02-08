package storage

import (
	"os"
	"strings"
	"testing"
)

func TestFileStorage_Save(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStorage(dir)
	email := Email{
		From:    "alice@example.com",
		To:      "bob@example.com",
		Content: "Hello, Bob!",
	}
	filename, err := fs.Save(email)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	// Check path format: should be <dir>/<to>/<from>/YYYY-MM-DD-HH-MM-SS.<nano>.txt
	expectedDir := dir + string(os.PathSeparator) + email.To + string(os.PathSeparator) + email.From
	if !strings.HasPrefix(filename, expectedDir) {
		t.Errorf("File not saved in correct directory. Got: %s, Want prefix: %s", filename, expectedDir)
	}
	// Check filename format
	parts := strings.Split(filename, string(os.PathSeparator))
	filePart := parts[len(parts)-1]
	if !strings.HasSuffix(filePart, ".txt") || len(strings.Split(filePart, ".")) < 3 {
		t.Errorf("Filename format incorrect: %s", filePart)
	}
	// Check file exists and content
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "alice@example.com") || !strings.Contains(content, "bob@example.com") || !strings.Contains(content, "Hello, Bob!") {
		t.Errorf("Saved file content incorrect: %s", content)
	}
}
