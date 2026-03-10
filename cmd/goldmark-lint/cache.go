package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mrueg/goldmark-lint/lint"
)

const cacheFileName = ".goldmark-lint-cache"

// cacheEntry stores the lint result for a single file indexed by its content hash.
type cacheEntry struct {
	Hash       string           `json:"hash"`
	ConfigHash string           `json:"configHash"`
	Violations []lint.Violation `json:"violations"`
}

// lintCache maps absolute file paths to their cached lint results.
type lintCache map[string]cacheEntry

// hashContent returns the SHA-256 hex digest of data.
func hashContent(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// hashConfig returns a SHA-256 hex digest that captures the effective rule
// configuration and the goldmark-lint version. encoding/json sorts map keys
// alphabetically, so the output is deterministic for equal configs.
// cfg originates from JSON/YAML config parsing and only contains
// JSON-compatible values, so json.Marshal will not return an error.
func hashConfig(cfg map[string]interface{}, ver string) string {
	data, _ := json.Marshal(cfg)
	combined := ver + ":" + string(data)
	return hashContent([]byte(combined))
}

// loadCache reads the cache file from dir and returns its contents.
// On any error an empty cache is returned.
func loadCache(dir string) lintCache {
	data, err := os.ReadFile(filepath.Join(dir, cacheFileName))
	if err != nil {
		return make(lintCache)
	}
	var c lintCache
	if err := json.Unmarshal(data, &c); err != nil {
		return make(lintCache)
	}
	return c
}

// saveCache writes c to the cache file in dir.
func saveCache(dir string, c lintCache) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, cacheFileName), data, 0644)
}
