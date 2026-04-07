package persist

import "github.com/phergul/apiscope/internal/model"

// LoadHistory loads the persisted history buckets from disk.
func (s *Store) LoadHistory() ([]model.PersistedHistoryBucket, error) {
	path, err := s.historyPath()
	if err != nil {
		return nil, err
	}

	var buckets []model.PersistedHistoryBucket
	if err := loadJSONFile(path, &buckets); err != nil {
		return nil, err
	}

	return buckets, nil
}

// SaveHistory writes the persisted history buckets to disk.
func (s *Store) SaveHistory(buckets []model.PersistedHistoryBucket) error {
	path, err := s.historyPath()
	if err != nil {
		return err
	}

	return saveJSONFile(path, buckets)
}
