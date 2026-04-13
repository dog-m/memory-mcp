package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const TIMESTAMP_FORMAT = time.RFC1123

type RecordID string

const RECORD_ID_NONE = RecordID("<invalid-id>")

type MemoryRecord struct {
	ID         RecordID `json:"id"`
	CreatedAt  string   `json:"created_at"`
	LastUpdate string   `json:"last_update"`
	Text       string   `json:"text"`
}

type Config struct {
	MaxMemories int `json:"max_memories"`
}

type MemoryStorage struct {
	mu          sync.RWMutex
	memories    map[RecordID]MemoryRecord
	dataPath    string
	maxMemories int
}

func StorageInit(dataPath string, maxMemories int) (*MemoryStorage, error) {
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, err
	}

	store := &MemoryStorage{
		memories:    make(map[RecordID]MemoryRecord),
		dataPath:    dataPath,
		maxMemories: maxMemories,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *MemoryStorage) load() error {
	filePath := filepath.Join(s.dataPath, "memories.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	records := make([]MemoryRecord, 0, s.maxMemories)
	if err := json.Unmarshal(data, &records); err != nil {
		return err
	}

	for _, v := range records {
		s.memories[v.ID] = v
	}

	return nil
}

func (s *MemoryStorage) save() error {
	records := make([]MemoryRecord, len(s.memories))
	i := 0
	for _, v := range s.memories {
		records[i] = v
		i++
	}

	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}

	filePath := filepath.Join(s.dataPath, "memories.json")
	return os.WriteFile(filePath, data, 0644)
}

func timestampToId(timestamp int64) RecordID {
	buff := make([]byte, 64/8)
	binary.LittleEndian.PutUint64(buff, uint64(timestamp))
	value := base64.URLEncoding.EncodeToString(buff)
	return RecordID(value[:11]) // truncating '=' padding at the end
}

func (s *MemoryStorage) NewRecord(text string) (RecordID, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	now_str := now.Format(TIMESTAMP_FORMAT)

	record := MemoryRecord{
		ID:         timestampToId(now.UnixMilli()),
		CreatedAt:  now_str,
		LastUpdate: now_str,
		Text:       text,
	}

	s.memories[record.ID] = record

	if err := s.save(); err != nil {
		return RECORD_ID_NONE, fmt.Errorf("failed to save memory: %w", err)
	}

	return record.ID, nil
}

func (s *MemoryStorage) GetAllRecords() []MemoryRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]MemoryRecord, 0, len(s.memories))
	for _, m := range s.memories {
		list = append(list, m)
	}
	return list
}

func (s *MemoryStorage) DeleteRecord(id RecordID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.memories[id]; !ok {
		return fmt.Errorf("memory record '%s' does not exist", id)
	}

	delete(s.memories, id)

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save after deletion: %w", err)
	}

	return nil
}

func (s *MemoryStorage) UpdateRecord(id RecordID, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := s.memories[id]
	if !ok {
		return fmt.Errorf("memory record '%s' does not exist", id)
	}

	record.Text = text
	record.LastUpdate = time.Now().Format(TIMESTAMP_FORMAT)
	s.memories[id] = record

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save after update: %w", err)
	}

	return nil
}
