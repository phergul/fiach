package theme

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed themes.json
var themesJSON []byte

type Tokens struct {
	Background         string `json:"background"`
	BackgroundElevated string `json:"backgroundElevated"`
	Surface            string `json:"surface"`
	SurfaceElevated    string `json:"surfaceElevated"`
	TextMuted          string `json:"textMuted"`
	TextSubtle         string `json:"textSubtle"`
	TextPrimary        string `json:"textPrimary"`
	Danger             string `json:"danger"`
	Warning            string `json:"warning"`
	AccentWarm         string `json:"accentWarm"`
	Info               string `json:"info"`
	Success            string `json:"success"`
	Accent             string `json:"accent"`
	BorderSubtle       string `json:"borderSubtle"`
	Border             string `json:"border"`
	BorderStrong       string `json:"borderStrong"`
	TagRed             string `json:"tagRed"`
	TagOrange          string `json:"tagOrange"`
	TagYellow          string `json:"tagYellow"`
	TagGreen           string `json:"tagGreen"`
	TagTeal            string `json:"tagTeal"`
	TagBlue            string `json:"tagBlue"`
	TagPurple          string `json:"tagPurple"`
	TagPink            string `json:"tagPink"`
}

type Definition struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Tokens Tokens `json:"tokens"`
}

var (
	definitions []Definition
	byID        map[string]Definition
)

func init() {
	if err := loadThemes(); err != nil {
		panic(err)
	}
}

func loadThemes() error {
	if err := json.Unmarshal(themesJSON, &definitions); err != nil {
		return fmt.Errorf("parse themes.json: %w", err)
	}

	if err := validateThemes(definitions); err != nil {
		return fmt.Errorf("validate themes.json: %w", err)
	}

	byID = make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		byID[definition.ID] = definition
	}

	return nil
}

func validateThemes(definitions []Definition) error {
	if len(definitions) == 0 {
		return fmt.Errorf("at least one theme is required")
	}

	seenIDs := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if strings.TrimSpace(definition.ID) == "" {
			return fmt.Errorf("theme id is required")
		}
		if strings.TrimSpace(definition.Label) == "" {
			return fmt.Errorf("theme %q: label is required", definition.ID)
		}
		if _, exists := seenIDs[definition.ID]; exists {
			return fmt.Errorf("duplicate theme id %q", definition.ID)
		}
		seenIDs[definition.ID] = struct{}{}

		if err := validateTokens(definition.ID, definition.Tokens); err != nil {
			return err
		}
	}

	return nil
}

func validateTokens(themeID string, tokens Tokens) error {
	for field, value := range map[string]string{
		"background":         tokens.Background,
		"backgroundElevated": tokens.BackgroundElevated,
		"surface":            tokens.Surface,
		"surfaceElevated":    tokens.SurfaceElevated,
		"textMuted":          tokens.TextMuted,
		"textSubtle":         tokens.TextSubtle,
		"textPrimary":        tokens.TextPrimary,
		"danger":             tokens.Danger,
		"warning":            tokens.Warning,
		"accentWarm":         tokens.AccentWarm,
		"info":               tokens.Info,
		"success":            tokens.Success,
		"accent":             tokens.Accent,
		"borderSubtle":       tokens.BorderSubtle,
		"border":             tokens.Border,
		"borderStrong":       tokens.BorderStrong,
		"tagRed":             tokens.TagRed,
		"tagOrange":          tokens.TagOrange,
		"tagYellow":          tokens.TagYellow,
		"tagGreen":           tokens.TagGreen,
		"tagTeal":            tokens.TagTeal,
		"tagBlue":            tokens.TagBlue,
		"tagPurple":          tokens.TagPurple,
		"tagPink":            tokens.TagPink,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("theme %q: token %q is required", themeID, field)
		}
	}

	return nil
}

func Definitions() []Definition {
	copied := make([]Definition, len(definitions))
	copy(copied, definitions)

	return copied
}

func Default() Definition {
	return definitions[0]
}

func Resolve(themeID string) Definition {
	themeID = strings.TrimSpace(themeID)
	if themeID == "" {
		return Default()
	}

	definition, ok := byID[themeID]
	if !ok {
		return Default()
	}

	return definition
}

func CSSColor(hex string) string {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) >= 6 {
		return "#" + hex[:6]
	}

	return "#" + hex
}
