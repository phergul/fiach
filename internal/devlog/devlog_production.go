//go:build production

package devlog

import "log/slog"

func SetLogger(*slog.Logger) {}

func SetEmitter(func(Entry)) {}

func Log(string) {}

func Logf(string, ...any) {}

func List(int) []Entry {
	return nil
}

func Clear() {}
