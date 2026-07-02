package theme

import "fmt"

func UpdaterCSS(themeID string) string {
	tokens := Resolve(themeID).Tokens
	primaryAction := CSSColor(tokens.Success)

	return fmt.Sprintf(`:root {
  color-scheme: dark;
  --bg: %s;
  --surface: %s;
  --surface-2: %s;
  --fg: %s;
  --fg-dim: %s;
  --fg-faint: %s;
  --border: %s;
  --accent: %s;
  --accent-fg: %s;
  --accent-dim: color-mix(in srgb, %s 24%%, transparent);
  --success: %s;
  --error: %s;
  --warning: %s;
  --radius: 0;
  --radius-sm: 0;
  --font: 'Atkinson Hyperlegible', ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, sans-serif;
}

.u__icon,
.u__btn,
.u__notes,
.u__bar,
.u__bar::-webkit-progress-bar,
.u__bar::-webkit-progress-value,
.u__bar::-moz-progress-bar,
.u__notes::-webkit-scrollbar-thumb {
  border-radius: 0;
}

.u__btn:focus-visible {
  outline: 2px solid %s;
  outline-offset: -2px;
}
`, CSSColor(tokens.Background),
		CSSColor(tokens.Surface),
		CSSColor(tokens.SurfaceElevated),
		CSSColor(tokens.TextPrimary),
		CSSColor(tokens.TextSubtle),
		CSSColor(tokens.TextMuted),
		CSSColor(tokens.Border),
		primaryAction,
		CSSColor(tokens.TextPrimary),
		primaryAction,
		CSSColor(tokens.Success),
		CSSColor(tokens.Danger),
		CSSColor(tokens.Warning),
		CSSColor(tokens.Info),
	)
}
