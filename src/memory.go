package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const TIMESTAMP_FORMAT = time.RFC1123

type MemoryRecord struct {
	ID         int64  `json:"id"`
	CreatedAt  string `json:"created_at"`
	LastUpdate string `json:"last_update"`
	Text       string `json:"text"`
}

type Config struct {
	MaxMemories int `json:"max_memories"`
}

type Store struct {
	mutex       sync.RWMutex
	memories    map[int64]MemoryRecord
	dataPath    string
	maxMemories int
}

func NewStore(dataPath string, maxMemories int) (*Store, error) {
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, err
	}

	store := &Store{
		memories:    make(map[int64]MemoryRecord),
		dataPath:    dataPath,
		maxMemories: maxMemories,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *Store) load() error {
	filePath := filepath.Join(s.dataPath, "memories.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &s.memories); err != nil {
		return err
	}

	return nil
}

func (s *Store) save() error {
	filePath := filepath.Join(s.dataPath, "memories.json")
	data, err := json.MarshalIndent(s.memories, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}

func (s *Store) Remember(text string) (int64, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if len(s.memories) >= s.maxMemories {
		return 0, errors.New("memory limit reached")
	}

	now := time.Now()
	now_str := now.Format(TIMESTAMP_FORMAT)
	id := now.UnixNano()
	_ = fmt.Sprintf("%X", id)

	record := MemoryRecord{
		ID:         id,
		CreatedAt:  now_str,
		LastUpdate: now_str,
		Text:       text,
	}

	s.memories[id] = record

	if err := s.save(); err != nil {
		return 0, fmt.Errorf("failed to save memory: %w", err)
	}

	return id, nil
}

func (s *Store) List() []MemoryRecord {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	list := make([]MemoryRecord, 0, len(s.memories))
	for _, m := range s.memories {
		list = append(list, m)
	}
	return list
}

func (s *Store) Forget(id int64) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, ok := s.memories[id]; !ok {
		return fmt.Errorf("memory record %d does not exist", id)
	}

	delete(s.memories, id)

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save after deletion: %w", err)
	}

	return nil
}

func (s *Store) Update(id int64, text string) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	record, ok := s.memories[id]
	if !ok {
		return fmt.Errorf("memory record %d does not exist", id)
	}

	record.Text = text
	record.LastUpdate = time.Now().Format(TIMESTAMP_FORMAT)
	s.memories[id] = record

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save after update: %w", err)
	}

	return nil
}
