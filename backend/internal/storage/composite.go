package storage

import "log"

// CompositeStorage writes to multiple storage backends simultaneously
type CompositeStorage struct {
	storages []Storage
}

// NewCompositeStorage creates a new composite storage with multiple backends
func NewCompositeStorage(storages ...Storage) *CompositeStorage {
	return &CompositeStorage{
		storages: storages,
	}
}

// Save saves to all configured storage backends
func (cs *CompositeStorage) Save(email Email) (string, error) {
	var results []string
	var lastErr error

	for _, storage := range cs.storages {
		filename, err := storage.Save(email)
		if err != nil {
			log.Printf("Error saving to storage: %v", err)
			lastErr = err
		} else {
			results = append(results, filename)
		}
	}

	// Return the first filename and any error that occurred
	if len(results) > 0 {
		return results[0], lastErr
	}

	if lastErr != nil {
		return "", lastErr
	}

	return "", nil
}

// Close closes all storage backends
func (cs *CompositeStorage) Close() error {
	for _, storage := range cs.storages {
		if closer, ok := storage.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}
