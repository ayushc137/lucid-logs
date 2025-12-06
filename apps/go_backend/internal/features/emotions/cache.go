package emotions

import (
	"context"
	"sync"

	"github.com/lucid-logs/go-backend/internal/shared/database"
	"github.com/rs/zerolog/log"
)

// =============================================================================
// EMOTION CACHE
// =============================================================================

// Cache holds all emotions in memory for fast lookups during inference.
// Loaded from DB at startup, refreshed on demand.
var (
	cache     map[string]*Emotion
	allList   []*Emotion
	cacheMu   sync.RWMutex
	cacheInit bool
)

// InitCache loads all emotions from the database into memory.
// Call this at application startup.
func InitCache(db *database.DB) error {
	repo := NewRepository(db)
	emotions, err := repo.GetAll(context.Background())
	if err != nil {
		return err
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	cache = make(map[string]*Emotion, len(emotions))
	allList = emotions

	for _, e := range emotions {
		// Store with full ID format only (consistent with categories:xxx pattern)
		cache[e.ID] = e // "emotions:E16"
	}

	cacheInit = true
	log.Info().Int("count", len(emotions)).Msg("emotion cache initialized")
	return nil
}

// GetByID retrieves an emotion from cache by ID.
// Returns nil if not found or cache not initialized.
func GetByID(id string) *Emotion {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	if !cacheInit {
		log.Warn().Msg("emotion cache not initialized")
		return nil
	}

	return cache[id]
}

// GetAll returns all emotions from cache.
func GetAll() []*Emotion {
	cacheMu.RLock()
	defer cacheMu.RUnlock()

	if !cacheInit {
		return nil
	}

	return allList
}

// IsValidEmotionID checks if an emotion ID exists in the cache.
func IsValidEmotionID(id string) bool {
	return GetByID(id) != nil
}

// IsCacheInitialized returns whether the cache has been loaded.
func IsCacheInitialized() bool {
	cacheMu.RLock()
	defer cacheMu.RUnlock()
	return cacheInit
}
