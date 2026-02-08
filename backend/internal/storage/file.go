package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type FileStorage struct {
	Dir string
}

func NewFileStorage(dir string) *FileStorage {
	os.MkdirAll(dir, 0755)
	return &FileStorage{Dir: dir}
}

func (fs *FileStorage) Save(email Email) (string, error) {
	now := time.Now()
	filename := fmt.Sprintf("%s.%09d.txt", now.Format("2006-01-02-15-04-05"), now.Nanosecond())
	dirPath := filepath.Join(fs.Dir, email.To, email.From)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dirPath, filename)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	content := fmt.Sprintf("From: %s\nTo: %s\n\n%s", email.From, email.To, email.Content)
	if _, err := f.WriteString(content); err != nil {
		return "", err
	}
	return path, nil
}
