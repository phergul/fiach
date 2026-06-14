export interface ThemeTokens {
  background: string;
  backgroundRgb: string;
  surface: string;
  surfaceRgb: string;
  surfaceElevated: string;
  textMuted: string;
  textSubtle: string;
  textPrimary: string;
  danger: string;
  dangerRgb: string;
  warning: string;
  accentWarm: string;
  info: string;
  success: string;
  accent: string;
  borderSubtle: string;
  border: string;
  borderStrong: string;
  tagRed: string;
  tagOrange: string;
  tagYellow: string;
  tagGreen: string;
  tagTeal: string;
  tagBlue: string;
  tagPurple: string;
  tagPink: string;
}

export interface ThemeDefinition {
  id: string;
  label: string;
  tokens: ThemeTokens;
}
