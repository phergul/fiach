import type { ThemeDefinition, ThemeTokens } from './themeTypes';

const createThemeTokens = (tokens: Omit<ThemeTokens, 'backgroundRgb' | 'surfaceRgb' | 'dangerRgb'> & {
  backgroundRgb?: string;
  dangerRgb?: string;
  surfaceRgb?: string;
}): ThemeTokens => ({
  ...tokens,
  backgroundRgb: tokens.backgroundRgb ?? toRGBChannels(tokens.background),
  dangerRgb: tokens.dangerRgb ?? toRGBChannels(tokens.danger),
  surfaceRgb: tokens.surfaceRgb ?? toRGBChannels(tokens.surface),
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

export const themes: ThemeDefinition[] = [
  {
    id: 'ash',
    label: 'Ash',
    tokens: createThemeTokens({
      background: '#252422ff',
      surface: '#2f2d2aff',
      surfaceElevated: '#403d39ff',
      textMuted: '#8f897fff',
      textSubtle: '#ccc5b9ff',
      textPrimary: '#fffcf2ff',
      danger: '#eb5e28ff',
      warning: '#d9823fff',
      accentWarm: '#f08a61ff',
      info: '#7d80daff',
      success: '#588b8bff',
      accent: '#7e9181ff',
      borderSubtle: '#34312dff',
      border: '#4d4943ff',
      borderStrong: '#625d55ff',
      shadowSubtle: '0 0.125rem 0.5rem rgb(0 0 0 / 18%)',
    }),
  },
  {
    id: 'spruce',
    label: 'Spruce',
    tokens: createThemeTokens({
      background: '#1e2421ff',
      surface: '#27302cff',
      surfaceElevated: '#33403bff',
      textMuted: '#95a39bff',
      textSubtle: '#c8d5ceff',
      textPrimary: '#f3faf6ff',
      danger: '#d96c59ff',
      warning: '#d8a050ff',
      accentWarm: '#cf8c63ff',
      info: '#7aa2d8ff',
      success: '#6aa48dff',
      accent: '#89b48fff',
      borderSubtle: '#303934ff',
      border: '#445048ff',
      borderStrong: '#5a6b61ff',
      shadowSubtle: '0 0.125rem 0.75rem rgb(3 10 7 / 20%)',
    }),
  },
  {
    id: 'midnight',
    label: 'Midnight',
    tokens: createThemeTokens({
      background: '#151a24ff',
      surface: '#1d2431ff',
      surfaceElevated: '#283244ff',
      textMuted: '#7f8ba3ff',
      textSubtle: '#bcc7deff',
      textPrimary: '#f6f8ffff',
      danger: '#f06a6aff',
      warning: '#d8a455ff',
      accentWarm: '#ef8c6aff',
      info: '#79a8f2ff',
      success: '#67b6a3ff',
      accent: '#7da0d6ff',
      borderSubtle: '#283141ff',
      border: '#364257ff',
      borderStrong: '#4a5a75ff',
      shadowSubtle: '0 0.125rem 0.75rem rgb(2 6 17 / 32%)',
    }),
  },
];

export const defaultTheme = themes[0];

export const themesByID = new Map(themes.map((theme) => [theme.id, theme]));

export const resolveTheme = (themeID: string | null | undefined) => {
  if (themeID === undefined || themeID === null || themeID.trim() === '') {
    return defaultTheme;
  }

  return themesByID.get(themeID) ?? defaultTheme;
};
