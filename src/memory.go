package main

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

type RecordID string

const TIMESTAMP_FORMAT = "2006-01-02 15:04:05 MST"
const RECORD_ID_NONE = RecordID("<invalid-id>")
const RECORD_FILENAME_FORMAT = "mem-%s.json"

var RECORD_FILENAME_MATCHER, _ = regexp.Compile("mem-[0-9A-Za-z_-]+.json")

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
	memories    map[RecordID]*MemoryRecord
	dataPath    string
	maxMemories int
}

func StorageInit(dataPath string, maxMemories int) (*MemoryStorage, error) {
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, err
	}

	storage := &MemoryStorage{
		memories:    make(map[RecordID]*MemoryRecord),
		dataPath:    dataPath,
		maxMemories: maxMemories,
	}

	if err := storage.loadAll(); err != nil {
		return nil, err
	}

	return storage, nil
}

func loadMemoryRecord(path string) (*MemoryRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	res := &MemoryRecord{}
	if err = json.NewDecoder(file).Decode(res); err != nil {
		return nil, err
	}

	return res, nil
}

func saveMemoryRecord(rec *MemoryRecord, dataPath string) error {
	data, err := json.MarshalIndent(rec, "", "    ")
	if err != nil {
		return err
	}

	path := fmt.Sprintf(RECORD_FILENAME_FORMAT, rec.ID)
	path = filepath.Join(dataPath, path)
	return os.WriteFile(path, data, 0644)
}

func (storage *MemoryStorage) loadAll() error {
	loaded := 0

	err := filepath.WalkDir(storage.dataPath, func(path string, info fs.DirEntry, err error) error {
		if err == nil {
			if !info.IsDir() && RECORD_FILENAME_MATCHER.MatchString(path) {
				if rec, err := loadMemoryRecord(path); err == nil {
					storage.memories[rec.ID] = rec
					loaded++
				}
			}
		}
		return nil
	})

	// TODO: unused? return?
	log.Printf("Loaded %d memory entries", loaded)
	return err
}

func getTimestamp() time.Time {
	return time.Now()
}

const (
	LCG_multiplier = uint64(0x5851f42d4c957f2d)
	LCG_const      = uint64(0x14057b7ef767814f)
)

func timestampToId(timestamp int64) RecordID {
	// simple Linear Congruential Generator
	id := uint64(timestamp)
	id = LCG_multiplier*id + LCG_const

	// convert
	buff := make([]byte, 64/8)
	binary.LittleEndian.PutUint64(buff, id)
	value := base64.URLEncoding.EncodeToString(buff)

	// truncate '=' padding at the end
	return RecordID(value[:11])
}

func (storage *MemoryStorage) AddRecord(text string) (*MemoryRecord, error) {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	now := getTimestamp()
	now_str := now.Format(TIMESTAMP_FORMAT)

	record := &MemoryRecord{
		ID:         timestampToId(now.UnixMilli()),
		CreatedAt:  now_str,
		LastUpdate: now_str,
		Text:       text,
	}

	if err := saveMemoryRecord(record, storage.dataPath); err != nil {
		return nil, fmt.Errorf("failed to save memory: %w", err)
	}

	// register it only after we were able to store it
	storage.memories[record.ID] = record

	return record, nil
}

func (storage *MemoryStorage) GetAllRecords() []*MemoryRecord {
	storage.mu.RLock()
	defer storage.mu.RUnlock()

	list := make([]*MemoryRecord, 0, len(storage.memories))
	for _, rec := range storage.memories {
		list = append(list, rec)
	}
	return list
}

func (storage *MemoryStorage) DeleteRecord(id RecordID) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	if _, ok := storage.memories[id]; !ok {
		return fmt.Errorf("memory record '%s' does not exist", id)
	}

	path := fmt.Sprintf(RECORD_FILENAME_FORMAT, id)
	path = filepath.Join(storage.dataPath, path)
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file entry for ID '%s': %w", id, err)
	}

	// updating collection only after successful file removal
	delete(storage.memories, id)

	return nil
}

func (storage *MemoryStorage) UpdateRecord(id RecordID, text string) error {
	storage.mu.Lock()
	defer storage.mu.Unlock()

	record, ok := storage.memories[id]
	if !ok {
		return fmt.Errorf("memory record '%s' does not exist", id)
	}

	// backup
	oldText, oldUpdate := record.Text, record.LastUpdate

	// update internal
	record.Text = text
	record.LastUpdate = getTimestamp().Format(TIMESTAMP_FORMAT)

	// update file
	if err := saveMemoryRecord(record, storage.dataPath); err != nil {
		// restore
		record.Text = oldText
		record.LastUpdate = oldUpdate

		return fmt.Errorf("failed to save after update: %w", err)
	}

	return nil
}
