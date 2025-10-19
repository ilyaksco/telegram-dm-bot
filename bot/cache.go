package bot

import (
	"sync"
	"time"
	"log"

	"telegram-dm-bot/storage"
)

const cacheDuration = 10 * time.Minute // Simpan cache selama 10 menit

type CacheEntry struct {
	channels  []storage.RegisteredChannel
	timestamp time.Time
}

type AdminCache struct {
	mu   sync.RWMutex
	data map[int64]CacheEntry
}

func NewAdminCache() *AdminCache {
	return &AdminCache{
		data: make(map[int64]CacheEntry),
	}
}

// Get mengambil data dari cache jika valid
func (c *AdminCache) Get(userID int64) ([]storage.RegisteredChannel, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, found := c.data[userID]
	if !found {
		return nil, false // Tidak ada di cache
	}

	if time.Since(entry.timestamp) > cacheDuration {
		return nil, false // Cache sudah kedaluwarsa
	}

	if entry.channels == nil {
		return nil, false 
	}

	return entry.channels, true // Cache valid
}

// Set menyimpan data ke cache
func (c *AdminCache) Set(userID int64, channels []storage.RegisteredChannel) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[userID] = CacheEntry{
		channels:  channels,
		timestamp: time.Now(),
	}
}
// --- AKHIR PERUBAHAN ---

func (c *AdminCache) Invalidate(userID int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, userID)
	log.Printf("cache invalidated for user %d", userID)
}