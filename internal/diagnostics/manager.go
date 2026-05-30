package diagnostics

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/phergul/fiach/internal/fileops"
)

const (
	DefaultLogFileName = "fiach.jsonl"
	DefaultMaxFileSize = 5 * 1024 * 1024
	DefaultMaxFiles    = 3

	DefaultRecentLimit = 100
	MaxRecentLimit     = 500
)

const (
	OperationScanGames      = "scan_games"
	OperationImportMod      = "import_mod"
	OperationApplyProfile   = "apply_profile"
	OperationRestoreVanilla = "restore_vanilla"
)

const (
	EventStarted   = "started"
	EventCompleted = "completed"
	EventFailed    = "failed"
)

type Options struct {
	LogPath     string
	MaxFileSize int64
	MaxFiles    int
}

type Manager struct {
	writer *rotatingWriter
	logger *slog.Logger

	subscribers      map[int]chan LogEntry
	subscribersLock  sync.RWMutex
	nextSubscriberID int
}

type RecentLogsInput struct {
	Limit     int
	Operation string
	Level     string
}

type LogEntry struct {
	Timestamp string
	Level     string
	Operation string
	Message   string
	Details   map[string]string
}

func NewManager(opts Options) (*Manager, error) {
	if strings.TrimSpace(opts.LogPath) == "" {
		return nil, errors.New("log path is required")
	}
	if opts.MaxFileSize <= 0 {
		opts.MaxFileSize = DefaultMaxFileSize
	}
	if opts.MaxFiles <= 0 {
		opts.MaxFiles = DefaultMaxFiles
	}

	writer, err := newRotatingWriter(opts.LogPath, opts.MaxFileSize, opts.MaxFiles)
	if err != nil {
		return nil, err
	}

	manager := &Manager{
		writer:      writer,
		subscribers: map[int]chan LogEntry{},
	}

	handler := slog.NewJSONHandler(managerLogWriter{manager: manager}, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	manager.logger = slog.New(handler)

	return manager, nil
}

func (m *Manager) Logger() *slog.Logger {
	if m == nil || m.logger == nil {
		return slog.Default()
	}

	return m.logger
}

func (m *Manager) Close() error {
	if m == nil || m.writer == nil {
		return nil
	}

	m.closeSubscribers()

	return m.writer.Close()
}

func (m *Manager) Subscribe() (<-chan LogEntry, func()) {
	if m == nil {
		closed := make(chan LogEntry)
		close(closed)
		return closed, func() {}
	}

	m.subscribersLock.Lock()
	defer m.subscribersLock.Unlock()

	m.nextSubscriberID++
	id := m.nextSubscriberID
	entries := make(chan LogEntry, 64)
	m.subscribers[id] = entries

	unsubscribe := func() {
		m.subscribersLock.Lock()
		defer m.subscribersLock.Unlock()

		if existing, ok := m.subscribers[id]; ok {
			delete(m.subscribers, id)
			close(existing)
		}
	}

	return entries, unsubscribe
}

func (m *Manager) RecentLogs(_ context.Context, input RecentLogsInput) ([]LogEntry, error) {
	if m == nil || m.writer == nil {
		return nil, errors.New("diagnostics manager is not configured")
	}

	limit := normalizedRecentLimit(input.Limit)

	files, err := m.writer.logFiles()
	if err != nil {
		return nil, err
	}

	entries := make([]LogEntry, 0, limit)
	for _, filePath := range files {
		if err := readLogFile(filePath, input, limit, &entries); err != nil {
			return nil, err
		}
	}

	reverseEntries(entries)
	return entries, nil
}

func (m *Manager) RecentRawLogs(_ context.Context, input RecentLogsInput) (lines []string, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("read recent raw diagnostics logs: %w", err)
		}
	}()

	if m == nil || m.writer == nil {
		return nil, errors.New("diagnostics manager is not configured")
	}

	limit := normalizedRecentLimit(input.Limit)

	files, err := m.writer.logFiles()
	if err != nil {
		return nil, err
	}

	lines = make([]string, 0, limit)
	for _, filePath := range files {
		if err := readRawLogFile(filePath, input, limit, &lines); err != nil {
			return nil, err
		}
	}

	reverseStrings(lines)
	return lines, nil
}

func PathLabel(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	clean := filepath.Clean(path)
	base := filepath.Base(clean)
	parent := filepath.Base(filepath.Dir(clean))
	if parent == "." || parent == string(filepath.Separator) || parent == "" {
		return base
	}

	return filepath.Join(parent, base)
}

func PathAttr(key string, path string) slog.Attr {
	return slog.String(key+"_label", PathLabel(path))
}

func ErrorAttr(err error) slog.Attr {
	if err == nil {
		return slog.String("error", "")
	}

	return slog.String("error", sanitizeDetail(err.Error()))
}

func DurationAttr(start time.Time) slog.Attr {
	return slog.Int64("duration_ms", time.Since(start).Milliseconds())
}

func readLogFile(filePath string, input RecentLogsInput, limit int, entries *[]LogEntry) error {
	file, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open diagnostics log %q: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		entry, ok := parseLogLine(scanner.Bytes())
		if !ok || !matchesFilters(entry, input) {
			continue
		}

		*entries = append(*entries, entry)
		if len(*entries) > limit {
			copy((*entries)[0:], (*entries)[1:])
			*entries = (*entries)[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read diagnostics log %q: %w", filePath, err)
	}

	return nil
}

func parseLogLine(line []byte) (LogEntry, bool) {
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return LogEntry{}, false
	}

	entry := LogEntry{
		Timestamp: stringValue(raw["time"]),
		Level:     strings.ToLower(stringValue(raw["level"])),
		Operation: stringValue(raw["operation"]),
		Message:   stringValue(raw["msg"]),
		Details:   map[string]string{},
	}
	if entry.Timestamp == "" || entry.Message == "" {
		return LogEntry{}, false
	}

	for key, value := range raw {
		if isReservedLogKey(key) || isUnsafeDetailKey(key) {
			continue
		}

		detail := stringValue(value)
		if detail == "" {
			continue
		}
		entry.Details[readableDetailKey(key)] = sanitizeDetail(detail)
	}

	return entry, true
}

func matchesFilters(entry LogEntry, input RecentLogsInput) bool {
	if input.Operation != "" && entry.Operation != input.Operation {
		return false
	}
	if input.Level != "" && !strings.EqualFold(entry.Level, input.Level) {
		return false
	}

	return true
}

func reverseEntries(entries []LogEntry) {
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
}

func (m *Manager) publish(entry LogEntry) {
	m.subscribersLock.RLock()
	defer m.subscribersLock.RUnlock()

	for _, subscriber := range m.subscribers {
		select {
		case subscriber <- entry:
		default:
		}
	}
}

func (m *Manager) closeSubscribers() {
	m.subscribersLock.Lock()
	defer m.subscribersLock.Unlock()

	for id, subscriber := range m.subscribers {
		delete(m.subscribers, id)
		close(subscriber)
	}
}

type managerLogWriter struct {
	manager *Manager
}

func (w managerLogWriter) Write(p []byte) (int, error) {
	if w.manager == nil || w.manager.writer == nil {
		return 0, errors.New("diagnostics manager is not configured")
	}

	n, err := w.manager.writer.Write(p)
	if err != nil {
		return n, err
	}

	if entry, ok := parseLogLine(p[:n]); ok {
		w.manager.publish(entry)
	}

	return n, nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprint(typed)
		}
		return string(encoded)
	}
}

func isReservedLogKey(key string) bool {
	switch key {
	case "time", "level", "msg", "operation":
		return true
	default:
		return false
	}
}

func isUnsafeDetailKey(key string) bool {
	key = strings.ToLower(key)
	if strings.HasSuffix(key, "_label") {
		return false
	}

	return strings.HasSuffix(key, "path") || strings.Contains(key, "_path")
}

func readableDetailKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "_", " ")
	key = strings.TrimSpace(key)
	if key == "" {
		return key
	}

	return strings.ToUpper(key[:1]) + key[1:]
}

func normalizedRecentLimit(limit int) int {
	if limit <= 0 {
		return DefaultRecentLimit
	}
	if limit > MaxRecentLimit {
		return MaxRecentLimit
	}

	return limit
}

func readRawLogFile(filePath string, input RecentLogsInput, limit int, lines *[]string) error {
	file, err := os.Open(filePath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open raw diagnostics log %q: %w", filePath, err)
	}
	defer func() {
		_ = file.Close()
	}()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		entry, ok := parseLogLine([]byte(line))
		if line == "" || !ok || !matchesFilters(entry, input) {
			continue
		}

		*lines = append(*lines, line)
		if len(*lines) > limit {
			copy((*lines)[0:], (*lines)[1:])
			*lines = (*lines)[:limit]
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read raw diagnostics log %q: %w", filePath, err)
	}

	return nil
}

func reverseStrings(values []string) {
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
}

func sanitizeDetail(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	value = quotedAbsolutePathPattern.ReplaceAllStringFunc(value, func(match string) string {
		return `"` + PathLabel(strings.Trim(match, `"`)) + `"`
	})
	value = unquotedAbsolutePathPattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := unquotedAbsolutePathPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}

		return parts[1] + PathLabel(parts[2])
	})

	return value
}

var quotedAbsolutePathPattern = regexp.MustCompile(`"(?:[A-Za-z]:\\|/)[^"]+"`)
var unquotedAbsolutePathPattern = regexp.MustCompile(`(^|[\s(])((?:[A-Za-z]:\\|/)[^\s:]+)`)

type rotatingWriter struct {
	mu          sync.Mutex
	path        string
	maxFileSize int64
	maxFiles    int
	file        *os.File
	size        int64
}

func newRotatingWriter(path string, maxFileSize int64, maxFiles int) (*rotatingWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create diagnostics log directory: %w", err)
	}

	writer := &rotatingWriter{
		path:        path,
		maxFileSize: maxFileSize,
		maxFiles:    maxFiles,
	}
	if err := writer.open(); err != nil {
		return nil, err
	}

	return writer, nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		if err := w.open(); err != nil {
			return 0, err
		}
	}
	if w.size > 0 && w.size+int64(len(p)) > w.maxFileSize {
		if err := w.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}

	err := w.file.Close()
	w.file = nil
	if err != nil {
		return fmt.Errorf("close diagnostics log: %w", err)
	}

	return nil
}

func (w *rotatingWriter) open() error {
	file, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open diagnostics log %q: %w", w.path, err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("stat diagnostics log %q: %w", w.path, err)
	}

	w.file = file
	w.size = info.Size()
	return nil
}

func (w *rotatingWriter) rotate() error {
	if w.file != nil {
		if err := w.file.Close(); err != nil {
			return fmt.Errorf("close diagnostics log before rotation: %w", err)
		}
		w.file = nil
	}

	if w.maxFiles <= 1 {
		if err := os.Remove(w.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove diagnostics log during rotation: %w", err)
		}
		return w.open()
	}

	oldest := rotatedPath(w.path, w.maxFiles-1)
	if err := os.Remove(oldest); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old diagnostics log %q: %w", oldest, err)
	}

	for index := w.maxFiles - 2; index >= 1; index-- {
		from := rotatedPath(w.path, index)
		to := rotatedPath(w.path, index+1)
		if err := fileops.RenameIfExists(from, to); err != nil {
			return err
		}
	}
	if err := fileops.RenameIfExists(w.path, rotatedPath(w.path, 1)); err != nil {
		return err
	}

	return w.open()
}

func (w *rotatingWriter) logFiles() ([]string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	files := make([]string, 0, w.maxFiles)
	for index := w.maxFiles - 1; index >= 1; index-- {
		path := rotatedPath(w.path, index)
		if exists, err := fileops.FileExists(path); err != nil {
			return nil, err
		} else if exists {
			files = append(files, path)
		}
	}
	if exists, err := fileops.FileExists(w.path); err != nil {
		return nil, err
	} else if exists {
		files = append(files, w.path)
	}

	return files, nil
}

func rotatedPath(path string, index int) string {
	return fmt.Sprintf("%s.%d", path, index)
}
