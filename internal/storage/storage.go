package storage

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const (
	defaultAppName = "mod-manager"
	databaseName   = "mod-manager.db"

	driverName      = "sqlite3"
	migrationsDir   = "migrations"
	filePermissions = 0755

	busyTimeoutMillis = "5000"
	sqliteJournalMode = "WAL"
	sqliteForeignKeys = "1"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

var gooseMu sync.Mutex

type Options struct {
	AppName string
	DataDir string
}

type Store struct {
	db   *sqlx.DB
	path string
}

func Open(ctx context.Context, opts Options) (*Store, error) {
	dbPath, err := databasePath(opts)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(dbPath), filePermissions); err != nil {
		return nil, fmt.Errorf("create database directory %q: %w", filepath.Dir(dbPath), err)
	}

	db, err := sqlx.Open(driverName, dataSourceName(dbPath))
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", dbPath, err)
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite database %q: %w", dbPath, err)
	}

	return &Store{
		db:   db,
		path: dbPath,
	}, nil
}

func (s *Store) DB() *sqlx.DB {
	return s.db
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	if err := s.db.Close(); err != nil {
		return fmt.Errorf("close sqlite database %q: %w", s.path, err)
	}

	return nil
}

func (s *Store) MigrateUp() error {
	if s == nil || s.db == nil {
		return errors.New("run migrations up: store is not open")
	}

	if err := runGoose(s.db.DB, goose.Up); err != nil {
		return fmt.Errorf("run migrations up: %w", err)
	}

	return nil
}

func (s *Store) MigrateDown() error {
	if s == nil || s.db == nil {
		return errors.New("run migrations down: store is not open")
	}

	if err := runGoose(s.db.DB, goose.Down); err != nil {
		return fmt.Errorf("run migrations down: %w", err)
	}

	return nil
}

func databasePath(opts Options) (string, error) {
	appName := opts.AppName
	if appName == "" {
		appName = defaultAppName
	}

	dataDir := opts.DataDir
	if dataDir == "" {
		dataDir = application.Path(application.PathDataHome)
	}
	if dataDir == "" {
		return "", errors.New("resolve database path: app data directory is empty")
	}

	return filepath.Join(dataDir, appName, databaseName), nil
}

func dataSourceName(dbPath string) string {
	values := url.Values{}
	values.Set("_busy_timeout", busyTimeoutMillis)
	values.Set("_foreign_keys", sqliteForeignKeys)
	values.Set("_journal_mode", sqliteJournalMode)

	return (&url.URL{
		Scheme:   "file",
		Path:     dbPath,
		RawQuery: values.Encode(),
	}).String()
}

func runGoose(db *sql.DB, fn func(*sql.DB, string, ...goose.OptionsFunc) error) error {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect(driverName); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	return fn(db, migrationsDir)
}

func gooseVersion(db *sql.DB) (int64, error) {
	gooseMu.Lock()
	defer gooseMu.Unlock()

	goose.SetLogger(goose.NopLogger())
	goose.SetBaseFS(migrationsFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect(driverName); err != nil {
		return 0, fmt.Errorf("set goose dialect: %w", err)
	}

	version, err := goose.GetDBVersion(db)
	if err != nil {
		return 0, fmt.Errorf("get goose version: %w", err)
	}

	return version, nil
}

func init() {
	goose.SetLogger(goose.NopLogger())
	goose.SetVerbose(false)
}
