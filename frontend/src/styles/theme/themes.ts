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
  {
    id: 'catppuccin-mocha',
    label: 'Catppuccin Mocha',
    tokens: createThemeTokens({
      background: '#1e1e2eff',
      surface: '#313244ff',
      surfaceElevated: '#45475aff',
      textMuted: '#7f849cff',
      textSubtle: '#bac2deff',
      textPrimary: '#cdd6f4ff',
      danger: '#f38ba8ff',
      warning: '#f9e2afff',
      accentWarm: '#fab387ff',
      info: '#89b4faff',
      success: '#a6e3a1ff',
      accent: '#cba6f7ff',
      borderSubtle: '#45475aff',
      border: '#6c7086ff',
      borderStrong: '#7f849cff',
      shadowSubtle: '0 0.125rem 0.75rem rgb(17 17 27 / 34%)',
    }),
  },
  {
    id: 'dracula',
    label: 'Dracula',
    tokens: createThemeTokens({
      background: '#282a36ff',
      surface: '#343746ff',
      surfaceElevated: '#44475aff',
      textMuted: '#8b8faeff',
      textSubtle: '#c5c8e8ff',
      textPrimary: '#f8f8f2ff',
      danger: '#ff5555ff',
      warning: '#f1fa8cff',
      accentWarm: '#ffb86cff',
      info: '#8be9fdff',
      success: '#50fa7bff',
      accent: '#bd93f9ff',
      borderSubtle: '#44475aff',
      border: '#6272a4ff',
      borderStrong: '#8b8faeff',
      shadowSubtle: '0 0.125rem 0.75rem rgb(25 26 36 / 35%)',
    }),
  },
  {
    id: 'nord',
    label: 'Nord',
    tokens: createThemeTokens({
      background: '#2e3440ff',
      surface: '#3b4252ff',
      surfaceElevated: '#434c5eff',
      textMuted: '#8f9aaeff',
      textSubtle: '#d8dee9ff',
      textPrimary: '#eceff4ff',
      danger: '#bf616aff',
      warning: '#ebcb8bff',
      accentWarm: '#d08770ff',
      info: '#81a1c1ff',
      success: '#a3be8cff',
      accent: '#88c0d0ff',
      borderSubtle: '#4c566aff',
      border: '#60708aff',
      borderStrong: '#81a1c1ff',
      shadowSubtle: '0 0.125rem 0.75rem rgb(20 24 31 / 32%)',
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
