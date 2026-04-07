package app

import "github.com/phergul/apiscope/internal/model"

const maxPersistedHistoryEntries = 15

func (s *Service) hydratePersistedState(result *LoadResult, rawSource string, apiSpec *model.APISpec) string {
	if s == nil || s.store == nil || result == nil || apiSpec == nil {
		return ""
	}

	notice := ""
	if err := s.syncRecentSpec(rawSource, apiSpec); err != nil && notice == "" {
		notice = persistenceUnavailableNotice
	}

	environments, err := s.loadEnvironments(result.Session.PersistenceScopeKey)
	if err != nil && notice == "" {
		notice = persistenceUnavailableNotice
	}
	result.Environments = environments

	history, maxRequestID, err := s.loadHistory(result.Session.PersistenceScopeKey, apiSpec)
	if err != nil && notice == "" {
		notice = persistenceUnavailableNotice
	}
	result.Session.RequestHistory = history
	result.Session.ActiveExecRequestID = maxRequestID
	result.View.ActiveExecuteRequestID = maxRequestID

	return notice
}

func (s *Service) loadHistory(scopeKey model.PersistenceScopeKey, apiSpec *model.APISpec) ([]model.HistoryEntry, uint64, error) {
	if s == nil || s.store == nil || scopeKey == "" || apiSpec == nil {
		return nil, 0, nil
	}

	buckets, err := s.store.LoadHistory()
	if err != nil {
		s.logger.Error("load history failed", "event", "persist_load_failed", "class", "history", "error", err.Error())
		return nil, 0, err
	}

	knownOperations := make(map[model.OperationKey]struct{}, len(apiSpec.Operations))
	for _, operation := range apiSpec.Operations {
		knownOperations[operation.Key] = struct{}{}
	}

	history := make([]model.HistoryEntry, 0)
	var maxRequestID uint64
	for _, bucket := range buckets {
		if bucket.ScopeKey != scopeKey {
			continue
		}
		if _, ok := knownOperations[bucket.OperationKey]; !ok {
			continue
		}
		for _, entry := range bucket.Entries {
			if _, ok := knownOperations[entry.OperationKey]; !ok {
				continue
			}
			history = append(history, rekeyHistoryEntry(cloneHistoryEntry(entry), apiSpec.Fingerprint))
			maxRequestID = max(maxRequestID, entry.RequestID)
		}
	}

	return history, maxRequestID, nil
}

// PersistHistoryEntry writes one execution history entry into durable history storage.
func (s *Service) PersistHistoryEntry(session model.SessionState, entry model.HistoryEntry) error {
	if s == nil || s.store == nil || session.PersistenceScopeKey == "" || entry.OperationKey == "" {
		return nil
	}

	buckets, err := s.store.LoadHistory()
	if err != nil {
		s.logger.Error("load history failed", "event", "persist_load_failed", "class", "history", "error", err.Error())
		return err
	}

	entry = cloneHistoryEntry(entry)
	bucketIndex := -1
	for index := range buckets {
		if buckets[index].ScopeKey == session.PersistenceScopeKey && buckets[index].OperationKey == entry.OperationKey {
			bucketIndex = index
			break
		}
	}

	if bucketIndex < 0 {
		buckets = append(buckets, model.PersistedHistoryBucket{
			ScopeKey:     session.PersistenceScopeKey,
			OperationKey: entry.OperationKey,
			Entries:      []model.HistoryEntry{entry},
		})
	} else {
		buckets[bucketIndex].Entries = append(buckets[bucketIndex].Entries, entry)
		buckets[bucketIndex].Entries = pruneHistoryEntries(buckets[bucketIndex].Entries)
	}

	if err := s.store.SaveHistory(buckets); err != nil {
		s.logger.Error("save history failed", "event", "persist_save_failed", "class", "history", "error", err.Error())
		return err
	}

	return nil
}

func pruneHistoryEntries(entries []model.HistoryEntry) []model.HistoryEntry {
	if len(entries) <= maxPersistedHistoryEntries {
		return entries
	}

	return append([]model.HistoryEntry(nil), entries[len(entries)-maxPersistedHistoryEntries:]...)
}

func rekeyHistoryEntry(entry model.HistoryEntry, specFingerprint model.SpecFingerprint) model.HistoryEntry {
	entry.Request.Draft.SpecFingerprint = specFingerprint
	entry.Request.Draft.Key = model.NewDraftKey(specFingerprint, entry.Request.Draft.OperationKey)
	return entry
}
