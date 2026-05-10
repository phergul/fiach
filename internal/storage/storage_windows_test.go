//go:build windows

package storage

import (
	"strings"
	"testing"
)

func TestDataSourceNameUsesWindowsFileURI(t *testing.T) {
	t.Parallel()

	dsn := dataSourceName(`C:\Users\Fergal\AppData\Local\mod-manager\mod-manager.db`)

	if !strings.HasPrefix(dsn, "file:///C:/Users/Fergal/AppData/Local/mod-manager/mod-manager.db?") {
		t.Fatalf("dataSourceName() = %q, want file URI with triple slash and slash-separated path", dsn)
	}
	if !strings.Contains(dsn, "_busy_timeout=5000") {
		t.Fatalf("dataSourceName() = %q, missing _busy_timeout", dsn)
	}
	if !strings.Contains(dsn, "_foreign_keys=1") {
		t.Fatalf("dataSourceName() = %q, missing _foreign_keys", dsn)
	}
	if !strings.Contains(dsn, "_journal_mode=WAL") {
		t.Fatalf("dataSourceName() = %q, missing _journal_mode", dsn)
	}
}
