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

type LocalRedis struct {
	mu       sync.RWMutex
	data     map[string]string
	sets     map[string]map[string]struct{}
	expiry   map[string]time.Time
	dataFile string
}

func NewLocalRedis(dataDir string) *LocalRedis {
	lr := &LocalRedis{
		data:     make(map[string]string),
		sets:     make(map[string]map[string]struct{}),
		expiry:   make(map[string]time.Time),
		dataFile: filepath.Join(dataDir, "redis_data.json"),
	}
	
	if err := os.MkdirAll(dataDir, 0755); err == nil {
		lr.loadFromFile()
	}
	
	go lr.expireLoop()
	return lr
}

type redisData struct {
	Data   map[string]string              `json:"data"`
	Sets   map[string]map[string]struct{} `json:"sets"`
	Expiry map[string]time.Time           `json:"expiry"`
}

func (r *LocalRedis) loadFromFile() {
	data, err := os.ReadFile(r.dataFile)
	if err != nil {
		return
	}
	
	var rd redisData
	if err := json.Unmarshal(data, &rd); err != nil {
		return
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if rd.Data != nil {
		r.data = rd.Data
	}
	if rd.Sets != nil {
		r.sets = rd.Sets
	}
	if rd.Expiry != nil {
		r.expiry = rd.Expiry
	}
}

func (r *LocalRedis) saveToFile() error {
	r.mu.RLock()
	rd := redisData{
		Data:   r.data,
		Sets:   r.sets,
		Expiry: r.expiry,
	}
	r.mu.RUnlock()
	
	data, err := json.Marshal(rd)
	if err != nil {
		return err
	}
	
	return os.WriteFile(r.dataFile, data, 0644)
}

func (r *LocalRedis) expireLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for key, expTime := range r.expiry {
			if now.After(expTime) {
				delete(r.data, key)
				delete(r.sets, key)
				delete(r.expiry, key)
			}
		}
		r.mu.Unlock()
	}
}

func (r *LocalRedis) Set(key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
	r.saveToFile()
	return nil
}

func (r *LocalRedis) Setex(key, value string, seconds int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[key] = value
	r.expiry[key] = time.Now().Add(time.Duration(seconds) * time.Second)
	r.saveToFile()
	return nil
}

func (r *LocalRedis) SetexCtx(ctx context.Context, key, value string, seconds int) error {
	return r.Setex(key, value, seconds)
}

func (r *LocalRedis) Get(key string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if exp, ok := r.expiry[key]; ok && time.Now().After(exp) {
		return "", errors.New("key expired")
	}
	
	val, ok := r.data[key]
	if !ok {
		return "", errors.New("key not found")
	}
	return val, nil
}

func (r *LocalRedis) GetCtx(ctx context.Context, key string) (string, error) {
	return r.Get(key)
}

func (r *LocalRedis) Del(keys ...string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	count := 0
	for _, key := range keys {
		if _, ok := r.data[key]; ok {
			delete(r.data, key)
			count++
		}
		if _, ok := r.sets[key]; ok {
			delete(r.sets, key)
			count++
		}
		delete(r.expiry, key)
	}
	r.saveToFile()
	return count, nil
}

func (r *LocalRedis) DelCtx(ctx context.Context, keys ...string) (int, error) {
	return r.Del(keys...)
}

func (r *LocalRedis) Exists(key string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if exp, ok := r.expiry[key]; ok && time.Now().After(exp) {
		return false, nil
	}
	
	_, ok1 := r.data[key]
	_, ok2 := r.sets[key]
	return ok1 || ok2, nil
}

func (r *LocalRedis) ExistsCtx(ctx context.Context, key string) (bool, error) {
	return r.Exists(key)
}

func (r *LocalRedis) Sadd(key string, members ...interface{}) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if r.sets[key] == nil {
		r.sets[key] = make(map[string]struct{})
	}
	
	count := 0
	for _, member := range members {
		memberStr := toString(member)
		if _, exists := r.sets[key][memberStr]; !exists {
			r.sets[key][memberStr] = struct{}{}
			count++
		}
	}
	r.saveToFile()
	return count, nil
}

func (r *LocalRedis) SaddCtx(ctx context.Context, key string, members ...interface{}) (int, error) {
	return r.Sadd(key, members...)
}

func (r *LocalRedis) Scard(key string) (int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if set, ok := r.sets[key]; ok {
		return int64(len(set)), nil
	}
	return 0, nil
}

func (r *LocalRedis) ScardCtx(ctx context.Context, key string) (int64, error) {
	return r.Scard(key)
}

func (r *LocalRedis) Sismember(key string, member interface{}) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if set, ok := r.sets[key]; ok {
		_, exists := set[toString(member)]
		return exists, nil
	}
	return false, nil
}

func (r *LocalRedis) SismemberCtx(ctx context.Context, key string, member interface{}) (bool, error) {
	return r.Sismember(key, member)
}

func (r *LocalRedis) Smembers(key string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if set, ok := r.sets[key]; ok {
		members := make([]string, 0, len(set))
		for member := range set {
			members = append(members, member)
		}
		return members, nil
	}
	return []string{}, nil
}

func (r *LocalRedis) SmembersCtx(ctx context.Context, key string) ([]string, error) {
	return r.Smembers(key)
}

func (r *LocalRedis) Srem(key string, members ...interface{}) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if set, ok := r.sets[key]; ok {
		count := 0
		for _, member := range members {
			memberStr := toString(member)
			if _, exists := set[memberStr]; exists {
				delete(set, memberStr)
				count++
			}
		}
		r.saveToFile()
		return count, nil
	}
	return 0, nil
}

func (r *LocalRedis) SremCtx(ctx context.Context, key string, members ...interface{}) (int, error) {
	return r.Srem(key, members...)
}

func (r *LocalRedis) Sscan(key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return r.SscanCtx(context.Background(), key, cursor, match, count)
}

func (r *LocalRedis) SscanCtx(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	if set, ok := r.sets[key]; ok {
		members := make([]string, 0, len(set))
		for member := range set {
			members = append(members, member)
		}
		
		start := int(cursor)
		if start >= len(members) {
			return []string{}, 0, nil
		}
		
		end := start + int(count)
		if end > len(members) {
			end = len(members)
		}
		
		result := members[start:end]
		nextCursor := uint64(0)
		if end < len(members) {
			nextCursor = uint64(end)
		}
		
		return result, nextCursor, nil
	}
	return []string{}, 0, nil
}

func (r *LocalRedis) Incr(key string) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var val int64 = 0
	if strVal, ok := r.data[key]; ok {
		var err error
		val, err = parseInt64(strVal)
		if err != nil {
			return 0, err
		}
	}
	val++
	r.data[key] = toString(val)
	r.saveToFile()
	return val, nil
}

func (r *LocalRedis) IncrCtx(ctx context.Context, key string) (int64, error) {
	return r.Incr(key)
}

func (r *LocalRedis) Expire(key string, seconds int) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.expiry[key] = time.Now().Add(time.Duration(seconds) * time.Second)
	r.saveToFile()
	return nil
}

func (r *LocalRedis) ExpireCtx(ctx context.Context, key string, seconds int) error {
	return r.Expire(key, seconds)
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	case int:
		return itoa(int64(val))
	case int64:
		return itoa(val)
	case float64:
		return itoa(int64(val))
	default:
		return ""
	}
}

func parseInt64(s string) (int64, error) {
	var val int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errors.New("not a number")
		}
		val = val*10 + int64(c-'0')
	}
	return val, nil
}

func itoa(i int64) string {
	if i == 0 {
		return "0"
	}
	
	negative := i < 0
	if negative {
		i = -i
	}
	
	var buf [20]byte
	pos := len(buf)
	
	for i > 0 {
		pos--
		buf[pos] = byte('0' + i%10)
		i /= 10
	}
	
	if negative {
		pos--
		buf[pos] = '-'
	}
	
	return string(buf[pos:])
}
