//go:build !production

package devlog

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

const maxEntries = 500

var (
	mu      sync.RWMutex
	entries []Entry
	emitter func(Entry)
	logger  *slog.Logger
)

func SetLogger(l *slog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	logger = l
}

func SetEmitter(fn func(Entry)) {
	mu.Lock()
	defer mu.Unlock()
	emitter = fn
}

func Log(message string) {
	appendEntry(message)
}

func Logf(format string, args ...any) {
	appendEntry(fmt.Sprintf(format, args...))
}

func List(limit int) []Entry {
	mu.RLock()
	defer mu.RUnlock()

	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}

	start := max(len(entries)-limit, 0)

	result := make([]Entry, limit)
	copy(result, entries[start:])
	return result
}

func Clear() {
	mu.Lock()
	defer mu.Unlock()
	entries = nil
}

func appendEntry(message string) {
	entry := Entry{
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Message:   message,
	}

	mu.Lock()
	entries = append(entries, entry)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	emit := emitter
	log := logger
	mu.Unlock()

	if log != nil {
		log.Debug(message)
	}
	if emit != nil {
		emit(entry)
	}
}
