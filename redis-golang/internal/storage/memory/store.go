package memory

import (
	"sync"
	"time"

	"redis_golang/pkg/logger"
)

// Obj represents a cached value and its expiration time
type Obj struct {
	Value     interface{}
	ExpiresAt int64 // Unix timestamp in milliseconds, -1 if no expiration
}

// Store is a thread-safe in-memory key-value store
type Store struct {
	mu   sync.RWMutex
	data map[string]*Obj
}

var globalStore *Store

func init() {
	globalStore = NewStore()
}

// NewStore initializes a new Store
func NewStore() *Store {
	return &Store{
		data: make(map[string]*Obj),
	}
}

// NewObj creates a new cache object
func NewObj(value interface{}, durationMs int64) *Obj {
	var expiresAt int64 = -1
	if durationMs > 0 {
		expiresAt = time.Now().UnixMilli() + durationMs
	}

	return &Obj{
		Value:     value,
		ExpiresAt: expiresAt,
	}
}

// Put adds or updates a key in the global store
func Put(k string, obj *Obj) {
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	globalStore.data[k] = obj
}

// Get retrieves a key from the global store
func Get(k string) *Obj {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	return globalStore.data[k]
}

// Del removes a key from the global store
func Del(k string) bool {
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	if _, exists := globalStore.data[k]; exists {
		delete(globalStore.data, k)
		return true
	}
	return false
}

// GetAllKeys returns all keys currently in the store
func GetAllKeys() []string {
	globalStore.mu.RLock()
	defer globalStore.mu.RUnlock()
	keys := make([]string, 0, len(globalStore.data))
	for k := range globalStore.data {
		keys = append(keys, k)
	}
	return keys
}

// CleanupExpiredKeys removes all expired keys from the store
func CleanupExpiredKeys() {
	now := time.Now().UnixMilli()
	
	globalStore.mu.Lock()
	defer globalStore.mu.Unlock()
	
	count := 0
	for key, obj := range globalStore.data {
		if obj.ExpiresAt != -1 && obj.ExpiresAt <= now {
			delete(globalStore.data, key)
			count++
		}
	}
	
	if count > 0 {
		logger.Log.Debug("cleaned up expired keys", "count", count)
	}
}

// StartCleanupRoutine starts a background goroutine that periodically cleans up expired keys
func StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Second) // Check every second
		defer ticker.Stop()
		
		for range ticker.C {
			CleanupExpiredKeys()
		}
	}()
	logger.Log.Info("started background cleanup routine for expired keys")
}
