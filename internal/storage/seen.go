package storage

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"time"
)

// Seen помнит ID уже отправленных вакансий, чтобы не слать одно и то же.
// Хранится в обычном JSON-файле — его коммитит обратно GitHub Actions.
type Seen struct {
	path string
	m    map[string]time.Time
}

// записи старше этого срока вычищаются: вакансия столько не живёт,
// а файл не должен расти вечно
const keepFor = 120 * 24 * time.Hour

func Load(path string) (*Seen, error) {
	s := &Seen{path: path, m: map[string]time.Time{}}

	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil // первого запуска файла ещё нет — это нормально
	}
	if err != nil {
		return nil, err
	}

	var raw map[string]time.Time
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	s.m = raw
	return s, nil
}

func (s *Seen) Has(id string) bool { return !s.m[id].IsZero() }

func (s *Seen) Add(id string) { s.m[id] = time.Now().UTC() }

func (s *Seen) Empty() bool { return len(s.m) == 0 }

func (s *Seen) Save() error {
	cutoff := time.Now().Add(-keepFor)
	for id, t := range s.m {
		if t.Before(cutoff) {
			delete(s.m, id)
		}
	}
	data, err := json.MarshalIndent(s.m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o644)
}
