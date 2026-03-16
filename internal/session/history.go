package session

import (
	"bufio"
	"encoding/json"
	"os"
	"syscall"
	"time"
)

// History manages append-only JSONL session history.
type History struct {
	Path string
}

// HistoryRecord is the on-disk JSON format (one per line).
type HistoryRecord struct {
	Type       string `json:"type"`
	ID         string `json:"id"`
	Workspace  string `json:"workspace,omitempty"`
	Tool       string `json:"tool,omitempty"`
	Time       string `json:"time"`
	Persistent bool   `json:"persistent,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

// HistoryEntry is the merged view returned by ListHistory.
type HistoryEntry struct {
	ID         string `json:"id"`
	Workspace  string `json:"workspace"`
	Tool       string `json:"tool"`
	Started    string `json:"started"`
	Stopped    string `json:"stopped,omitempty"`
	Persistent bool   `json:"persistent"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

// RecordStart appends a start record.
func (h *History) RecordStart(id, workspace, tool string, persistent bool) error {
	rec := HistoryRecord{
		Type:       "start",
		ID:         id,
		Workspace:  workspace,
		Tool:       tool,
		Time:       time.Now().UTC().Format(time.RFC3339),
		Persistent: persistent,
	}
	return h.appendRecord(rec)
}

// RecordStop appends a stop record.
func (h *History) RecordStop(id string, exitCode int) error {
	rec := HistoryRecord{
		Type:     "stop",
		ID:       id,
		Time:     time.Now().UTC().Format(time.RFC3339),
		ExitCode: exitCode,
	}
	return h.appendRecord(rec)
}

func (h *History) appendRecord(rec HistoryRecord) error {
	f, err := os.OpenFile(h.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(f.Fd()), syscall.LOCK_UN)

	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	_, err = f.Write(append(data, '\n'))
	return err
}

// ListHistory reads and merges start/stop records, newest first.
func (h *History) ListHistory(limit, offset int) ([]HistoryEntry, error) {
	f, err := os.Open(h.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	starts := make(map[string]*HistoryEntry)
	var order []string

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var rec HistoryRecord
		if err := json.Unmarshal(scanner.Bytes(), &rec); err != nil {
			continue
		}
		switch rec.Type {
		case "start":
			entry := &HistoryEntry{
				ID:         rec.ID,
				Workspace:  rec.Workspace,
				Tool:       rec.Tool,
				Started:    rec.Time,
				Persistent: rec.Persistent,
			}
			starts[rec.ID] = entry
			order = append(order, rec.ID)
		case "stop":
			if e, ok := starts[rec.ID]; ok {
				e.Stopped = rec.Time
				e.ExitCode = rec.ExitCode
			}
		}
	}

	// Reverse to show newest first
	result := make([]HistoryEntry, 0, len(order))
	for i := len(order) - 1; i >= 0; i-- {
		result = append(result, *starts[order[i]])
	}

	// Apply offset and limit
	if offset >= len(result) {
		return nil, nil
	}
	result = result[offset:]
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}
