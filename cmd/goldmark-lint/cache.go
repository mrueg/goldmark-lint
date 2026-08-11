package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mrueg/goldmark-lint/lint"
)

const cacheFileName = ".goldmark-lint-cache"

// maxCacheEntryAge is how long an entry survives without being used. Entries
// are evicted by age rather than by count so that a genuinely large repository
// (the tldr-pages corpus used by bench/bench.sh has ~34,000 files) stays fully
// cached, while results from one-off runs over directories that are no longer
// linted are eventually reclaimed.
const maxCacheEntryAge = 30 * 24 * time.Hour

// lastUsedRefreshInterval is how stale an entry's LastUsed stamp may get before
// a run that only produced cache hits still rewrites the cache file. Without
// this, refreshing timestamps would rewrite the whole cache on every run; with
// it, a fully-cached repository is rewritten at most once a day.
const lastUsedRefreshInterval = 24 * time.Hour

// cacheEntry stores the lint result for a single file indexed by its content hash.
type cacheEntry struct {
	Hash       string `json:"hash"`
	ConfigHash string `json:"configHash"`
	// LastUsed is the Unix time in seconds at which this entry was last read or
	// written. Entries loaded from a cache written by an older version have no
	// stamp; pruneCache treats those as used now so that upgrading does not
	// discard an otherwise valid cache.
	LastUsed   int64            `json:"lastUsed,omitempty"`
	Violations []lint.Violation `json:"violations"`
}

// lintCache maps absolute file paths to their cached lint results.
type lintCache map[string]cacheEntry

// pruneCache removes entries that can no longer be useful. An entry is dropped
// when its file no longer exists, or when it has not been used within
// maxCacheEntryAge. Paths in touched were used by the current run and are
// always kept. It returns how many entries were removed and how many survivors
// were given a LastUsed stamp they were missing.
//
// Without this the cache only ever grew: entries were merged in and never
// removed, so a checkout that had run the benchmark corpora accumulated a
// 16 MB cache holding 42,000 entries, most of them for files outside any
// normal run, which then had to be parsed on every invocation.
func pruneCache(c lintCache, touched map[string]bool, now time.Time) (removed, stamped int) {
	cutoff := now.Add(-maxCacheEntryAge).Unix()
	for path, entry := range c {
		if touched[path] {
			continue
		}
		if entry.LastUsed != 0 && entry.LastUsed < cutoff {
			delete(c, path)
			removed++
			continue
		}
		if _, err := os.Stat(path); err != nil {
			delete(c, path)
			removed++
			continue
		}
		// A missing stamp means the entry was written by a version predating
		// LastUsed. Keep it — upgrading should not throw away a valid cache —
		// but stamp it now so that it ages out normally from here on rather
		// than being grandfathered in forever.
		if entry.LastUsed == 0 {
			entry.LastUsed = now.Unix()
			c[path] = entry
			stamped++
		}
	}
	return removed, stamped
}

// needsTimestampRefresh reports whether any entry used by this run carries a
// LastUsed stamp old enough to be worth rewriting the cache for.
func needsTimestampRefresh(c lintCache, touched map[string]bool, now time.Time) bool {
	threshold := now.Add(-lastUsedRefreshInterval).Unix()
	for path := range touched {
		if entry, ok := c[path]; ok && entry.LastUsed < threshold {
			return true
		}
	}
	return false
}

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
