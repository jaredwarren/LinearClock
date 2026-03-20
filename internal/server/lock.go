package server

import (
	"sync"

	"github.com/jaredwarren/clock/internal/config"
)

// configWriteMu serializes concurrent config.gob writes from:
// - HTTP handlers (UpdateConfig / UpdateEvents / delete/edit/move)
// - the iCal sync loop in cmd/configd
//
// Note: internal/config.WriteConfig already writes atomically (temp + rename),
// but serializing writers still avoids last-writer-wins surprises during updates.
var configWriteMu sync.Mutex

// WriteConfigLocked writes cfg to filepath under a shared mutex.
func WriteConfigLocked(filepath string, c *config.Config) error {
	configWriteMu.Lock()
	defer configWriteMu.Unlock()
	return config.WriteConfig(filepath, c)
}

