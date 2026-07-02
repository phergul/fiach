import themeDefinitions from '@fiach/theme/themes.json';

import type { ThemeDefinition, ThemeTokens } from './themeTypes';

type RawThemeTokens = Omit<
  ThemeTokens,
  'backgroundRgb' | 'backgroundElevatedRgb' | 'surfaceRgb' | 'dangerRgb'
>;

type RawThemeDefinition = {
  id: string;
  label: string;
  tokens: RawThemeTokens;
};

const createThemeTokens = (tokens: RawThemeTokens): ThemeTokens => ({
  ...tokens,
  backgroundRgb: toRGBChannels(tokens.background),
  backgroundElevatedRgb: toRGBChannels(tokens.backgroundElevated),
  dangerRgb: toRGBChannels(tokens.danger),
  surfaceRgb: toRGBChannels(tokens.surface),
});

const toRGBChannels = (value: string) => {
  const hex = value.replace('#', '');
  if (hex.length !== 6 && hex.length !== 8) {
    throw new Error(`Unsupported theme color ${value}`);
  }

  const channels = [hex.slice(0, 2), hex.slice(2, 4), hex.slice(4, 6)].map((channel) =>
    Number.parseInt(channel, 16),
  );

  return channels.join(' ');
};

const rawThemes = themeDefinitions as RawThemeDefinition[];

export const themes: ThemeDefinition[] = rawThemes.map((theme) => ({
  id: theme.id,
  label: theme.label,
  tokens: createThemeTokens(theme.tokens),
}));

export const defaultTheme = themes[0];

export const themesByID = new Map(themes.map((theme) => [theme.id, theme]));

export const resolveTheme = (themeID: string | null | undefined) => {
  if (themeID === undefined || themeID === null || themeID.trim() === '') {
    return defaultTheme;
  }

  return themesByID.get(themeID) ?? defaultTheme;
};
