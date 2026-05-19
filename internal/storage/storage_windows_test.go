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
	if !strings.Contains(dsn, "_pragma=busy_timeout%3D5000") {
		t.Fatalf("dataSourceName() = %q, missing busy_timeout pragma", dsn)
	}
	if !strings.Contains(dsn, "_pragma=foreign_keys%3D1") {
		t.Fatalf("dataSourceName() = %q, missing foreign_keys pragma", dsn)
	}
	if !strings.Contains(dsn, "_pragma=journal_mode%28WAL%29") {
		t.Fatalf("dataSourceName() = %q, missing journal_mode pragma", dsn)
	}
}
