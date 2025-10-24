package localstorage

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrNotFound      = errors.New("document not found")
	ErrAlreadyExists = errors.New("document already exists")
)

type Document interface {
	GetID() string
	SetID(id string)
}

type Collection struct {
	mu       sync.RWMutex
	name     string
	data     map[string]interface{}
	dataFile string
}

func NewCollection(name string, dataDir string) *Collection {
	c := &Collection{
		name:     name,
		data:     make(map[string]interface{}),
		dataFile: filepath.Join(dataDir, name+".json"),
	}
	
	if err := os.MkdirAll(dataDir, 0755); err == nil {
		c.loadFromFile()
	}
	
	return c
}

func (c *Collection) loadFromFile() {
	data, err := os.ReadFile(c.dataFile)
	if err != nil {
		return
	}
	
	c.mu.Lock()
	defer c.mu.Unlock()
	
	json.Unmarshal(data, &c.data)
}

func (c *Collection) saveToFile() error {
	c.mu.RLock()
	data, err := json.Marshal(c.data)
	c.mu.RUnlock()
	
	if err != nil {
		return err
	}
	
	return os.WriteFile(c.dataFile, data, 0644)
}

func (c *Collection) Insert(ctx context.Context, id string, doc interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.data[id]; exists {
		return ErrAlreadyExists
	}
	
	c.data[id] = doc
	c.saveToFile()
	return nil
}

func (c *Collection) FindOne(ctx context.Context, id string, result interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	doc, ok := c.data[id]
	if !ok {
		return ErrNotFound
	}
	
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, result)
}

func (c *Collection) Update(ctx context.Context, id string, doc interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.data[id]; !exists {
		return ErrNotFound
	}
	
	c.data[id] = doc
	c.saveToFile()
	return nil
}

func (c *Collection) Upsert(ctx context.Context, id string, doc interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data[id] = doc
	c.saveToFile()
	return nil
}

func (c *Collection) Delete(ctx context.Context, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if _, exists := c.data[id]; !exists {
		return ErrNotFound
	}
	
	delete(c.data, id)
	c.saveToFile()
	return nil
}

func (c *Collection) FindAll(ctx context.Context, filter func(interface{}) bool, result interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	var matches []interface{}
	for _, doc := range c.data {
		if filter == nil || filter(doc) {
			matches = append(matches, doc)
		}
	}
	
	data, err := json.Marshal(matches)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, result)
}

func (c *Collection) Count(ctx context.Context, filter func(interface{}) bool) (int, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	count := 0
	for _, doc := range c.data {
		if filter == nil || filter(doc) {
			count++
		}
	}
	
	return count, nil
}

func (c *Collection) Drop(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.data = make(map[string]interface{})
	return os.Remove(c.dataFile)
}

type LocalStorage struct {
	mu          sync.RWMutex
	collections map[string]*Collection
	dataDir     string
}

func NewLocalStorage(dataDir string) *LocalStorage {
	if dataDir == "" {
		dataDir = "./data"
	}
	
	os.MkdirAll(dataDir, 0755)
	
	return &LocalStorage{
		collections: make(map[string]*Collection),
		dataDir:     dataDir,
	}
}

func (s *LocalStorage) Collection(name string) *Collection {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if c, ok := s.collections[name]; ok {
		return c
	}
	
	c := NewCollection(name, s.dataDir)
	s.collections[name] = c
	return c
}

type QueryHelper struct{}

func (q *QueryHelper) MatchString(field, value string, doc interface{}) bool {
	data, _ := json.Marshal(doc)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	
	if v, ok := m[field]; ok {
		if str, ok := v.(string); ok {
			return str == value
		}
	}
	return false
}

func (q *QueryHelper) MatchTime(field string, start, end time.Time, doc interface{}) bool {
	data, _ := json.Marshal(doc)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	
	if v, ok := m[field]; ok {
		if timeStr, ok := v.(string); ok {
			t, err := time.Parse(time.RFC3339, timeStr)
			if err == nil {
				return !t.Before(start) && !t.After(end)
			}
		}
	}
	return false
}

func (q *QueryHelper) MatchInSlice(field string, values []string, doc interface{}) bool {
	data, _ := json.Marshal(doc)
	var m map[string]interface{}
	json.Unmarshal(data, &m)
	
	if v, ok := m[field]; ok {
		if str, ok := v.(string); ok {
			for _, val := range values {
				if str == val {
					return true
				}
			}
		}
	}
	return false
}
