package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type cacheEnvelope struct {
	ExpiresAt time.Time       `json:"expires_at"`
	Data      json.RawMessage `json:"data"`
}

// PersistentCache provides a two-tier in-memory + disk cache with Stale-While-Revalidate support
type PersistentCache struct {
	mu         sync.RWMutex
	dir        string
	defaultTTL time.Duration
	memory     map[string]cacheEnvelope
}

// NewPersistentCache creates a new PersistentCache stored in dir with defaultTTL
func NewPersistentCache(dir string, defaultTTL time.Duration) (*PersistentCache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache dir: %w", err)
	}

	return &PersistentCache{
		dir:        dir,
		defaultTTL: defaultTTL,
		memory:     make(map[string]cacheEnvelope),
	}, nil
}

func (c *PersistentCache) keyToPath(key string) string {
	h := sha256.Sum256([]byte(key))
	filename := hex.EncodeToString(h[:]) + ".json"
	return filepath.Join(c.dir, filename)
}

// Get retrieves an item into target, indicating if it was found and if it is stale
func (c *PersistentCache) Get(key string, target interface{}) (found bool, stale bool) {
	c.mu.RLock()
	envelope, ok := c.memory[key]
	c.mu.RUnlock()

	// If not in memory, check disk
	if !ok {
		filePath := c.keyToPath(key)
		data, err := os.ReadFile(filePath)
		if err == nil {
			var diskEnvelope cacheEnvelope
			if err := json.Unmarshal(data, &diskEnvelope); err == nil {
				envelope = diskEnvelope
				ok = true
				// Populate back into L1 memory cache
				c.mu.Lock()
				c.memory[key] = envelope
				c.mu.Unlock()
			}
		}
	}

	if !ok {
		return false, false
	}

	isStale := time.Now().After(envelope.ExpiresAt)
	if target != nil {
		if err := json.Unmarshal(envelope.Data, target); err != nil {
			return false, false
		}
	}

	return true, isStale
}

// Set stores an item in both memory and disk
func (c *PersistentCache) Set(key string, val interface{}) error {
	bytes, err := json.Marshal(val)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	envelope := cacheEnvelope{
		ExpiresAt: time.Now().Add(c.defaultTTL),
		Data:      bytes,
	}

	c.mu.Lock()
	c.memory[key] = envelope
	c.mu.Unlock()

	// Persist to disk asynchronously or synchronously
	filePath := c.keyToPath(key)
	diskBytes, err := json.Marshal(envelope)
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, diskBytes, 0644)
}

// GetOrFetch retrieves from cache or fetches, with Stale-While-Revalidate fallback
func (c *PersistentCache) GetOrFetch(key string, target interface{}, fetcher func() (interface{}, error)) error {
	found, stale := c.Get(key, target)

	if found && !stale {
		return nil
	}

	// Stale or not found: call fetcher
	newVal, err := fetcher()
	if err != nil {
		// If we have stale data, serve it gracefully!
		if found {
			return nil
		}
		return err
	}

	// Fetcher succeeded: update cache
	if err := c.Set(key, newVal); err != nil {
		return err
	}

	// Unmarshal newVal into target
	bytes, _ := json.Marshal(newVal)
	return json.Unmarshal(bytes, target)
}
