import type { ThemeDefinition, ThemeTokens } from './themeTypes';

const themeTokenVariableEntries = (
  tokens: ThemeTokens,
): Array<[property: string, value: string]> => [
  ['--color-background', tokens.background],
  ['--color-background-rgb', tokens.backgroundRgb],
  ['--color-surface', tokens.surface],
  ['--color-surface-rgb', tokens.surfaceRgb],
  ['--color-surface-elevated', tokens.surfaceElevated],
  ['--color-text-muted', tokens.textMuted],
  ['--color-text-subtle', tokens.textSubtle],
  ['--color-text-primary', tokens.textPrimary],
  ['--color-danger', tokens.danger],
  ['--color-danger-rgb', tokens.dangerRgb],
  ['--color-warning', tokens.warning],
  ['--color-accent-warm', tokens.accentWarm],
  ['--color-info', tokens.info],
  ['--color-success', tokens.success],
  ['--color-accent', tokens.accent],
  ['--color-border-subtle', tokens.borderSubtle],
  ['--color-border', tokens.border],
  ['--color-border-strong', tokens.borderStrong],
  ['--color-tag-red', tokens.tagRed],
  ['--color-tag-orange', tokens.tagOrange],
  ['--color-tag-yellow', tokens.tagYellow],
  ['--color-tag-green', tokens.tagGreen],
  ['--color-tag-teal', tokens.tagTeal],
  ['--color-tag-blue', tokens.tagBlue],
  ['--color-tag-purple', tokens.tagPurple],
  ['--color-tag-pink', tokens.tagPink],
];

export const applyThemeCSSVariables = (
  theme: ThemeDefinition,
  root: HTMLElement = document.documentElement,
) => {
  root.dataset.theme = theme.id;

  for (const [property, value] of themeTokenVariableEntries(theme.tokens)) {
    root.style.setProperty(property, value);
  }
};
