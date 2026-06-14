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
      tagRed: '#e06c75ff',
      tagOrange: '#d9823fff',
      tagYellow: '#d6b95cff',
      tagGreen: '#83a65fff',
      tagTeal: '#588b8bff',
      tagBlue: '#6f91caff',
      tagPurple: '#9b7fc9ff',
      tagPink: '#cf779eff',
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
      tagRed: '#d96c59ff',
      tagOrange: '#cf8c63ff',
      tagYellow: '#d8b864ff',
      tagGreen: '#89b48fff',
      tagTeal: '#6aa48dff',
      tagBlue: '#7aa2d8ff',
      tagPurple: '#a58ac7ff',
      tagPink: '#cf829fff',
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
      tagRed: '#f06a6aff',
      tagOrange: '#ef8c6aff',
      tagYellow: '#d8a455ff',
      tagGreen: '#79b980ff',
      tagTeal: '#67b6a3ff',
      tagBlue: '#79a8f2ff',
      tagPurple: '#a68ce0ff',
      tagPink: '#df7fa8ff',
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
      borderSubtle: '#3b3d52ff',
      border: '#585b70ff',
      borderStrong: '#6c7086ff',
      tagRed: '#f38ba8ff',
      tagOrange: '#fab387ff',
      tagYellow: '#f9e2afff',
      tagGreen: '#a6e3a1ff',
      tagTeal: '#94e2d5ff',
      tagBlue: '#89b4faff',
      tagPurple: '#cba6f7ff',
      tagPink: '#f5c2e7ff',
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
      borderSubtle: '#3f4255ff',
      border: '#565b74ff',
      borderStrong: '#717699ff',
      tagRed: '#ff5555ff',
      tagOrange: '#ffb86cff',
      tagYellow: '#f1fa8cff',
      tagGreen: '#50fa7bff',
      tagTeal: '#8be9d0ff',
      tagBlue: '#8be9fdff',
      tagPurple: '#bd93f9ff',
      tagPink: '#ff79c6ff',
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
      borderSubtle: '#434c5eff',
      border: '#4c566aff',
      borderStrong: '#60708aff',
      tagRed: '#bf616aff',
      tagOrange: '#d08770ff',
      tagYellow: '#ebcb8bff',
      tagGreen: '#a3be8cff',
      tagTeal: '#8fbcbbff',
      tagBlue: '#81a1c1ff',
      tagPurple: '#b48eadff',
      tagPink: '#d0879dff',
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
