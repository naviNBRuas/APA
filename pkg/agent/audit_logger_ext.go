package agent

import (
	"bufio"
	"encoding/json"
	"os"
)

// ReadRecent returns up to n most recent audit entries from the log file.
func (al *AuditLogger) ReadRecent(n int) ([]AuditEntry, error) {
	al.mu.Lock()
	defer al.mu.Unlock()

	f, err := os.Open(al.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var entries []AuditEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry AuditEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			entries = append(entries, entry)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}
	return entries, nil
}
