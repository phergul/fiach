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
  shadowSubtle: string;
}

export interface ThemeDefinition {
  id: string;
  label: string;
  tokens: ThemeTokens;
}
