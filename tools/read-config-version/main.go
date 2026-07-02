package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	data, err := os.ReadFile("build/config.yml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "read build/config.yml: %v\n", err)
		os.Exit(1)
	}

	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	infoStart := strings.Index(text, "info:\n")
	if infoStart < 0 {
		fmt.Fprintln(os.Stderr, "info block not found in build/config.yml")
		os.Exit(1)
	}

	infoText := text[infoStart:]
	if end := strings.Index(infoText, "\n\n"); end >= 0 {
		infoText = infoText[:end]
	}

	match := regexp.MustCompile(`(?m)^  version: "([^"]+)"`).FindStringSubmatch(infoText)
	if len(match) < 2 {
		fmt.Fprintln(os.Stderr, "info.version not found in build/config.yml")
		os.Exit(1)
	}

	fmt.Print(match[1])
}
