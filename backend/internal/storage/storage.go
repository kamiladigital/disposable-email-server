package storage

// Email represents a simple email structure
// (expand as needed for more fields)
type Email struct {
	From    string
	To      string
	Content string
}

// Storage is the interface for saving emails
// Save should return the filename or an error
type Storage interface {
	Save(email Email) (string, error)
}
