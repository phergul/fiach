import type { ReactNode } from 'react';
import { createContext, useContext, useEffect, useState } from 'react';

import {
  GetThemeID,
  SetThemeID,
} from '@bindings/github.com/phergul/fiach/internal/services/settingsservice';
import { applyThemeCSSVariables } from '@theme/themeCSSVariables';
import { defaultTheme, resolveTheme, themes } from '@theme/themes';
import type { ThemeDefinition } from '@theme/themeTypes';

interface ThemeContextValue {
  activeTheme: ThemeDefinition;
  isLoading: boolean;
  setTheme: (themeID: string) => Promise<void>;
  themes: ThemeDefinition[];
}

const ThemeContext = createContext<ThemeContextValue | null>(null);

interface ThemeProviderProps {
  children: ReactNode;
}

export const ThemeProvider = ({ children }: ThemeProviderProps) => {
  const [activeTheme, setActiveTheme] = useState(defaultTheme);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    let isMounted = true;

    const loadTheme = async () => {
      try {
        const storedThemeID = await GetThemeID();
        if (!isMounted) {
          return;
        }

        setActiveTheme(resolveTheme(storedThemeID));
      } catch (error) {
        console.error('Failed to load theme setting', error);
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    void loadTheme();

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    applyThemeCSSVariables(activeTheme);
  }, [activeTheme]);

  const contextValue: ThemeContextValue = {
    activeTheme,
    isLoading,
    setTheme: async (themeID: string) => {
      await SetThemeID(themeID);
      setActiveTheme(resolveTheme(themeID));
    },
    themes,
  };

  return <ThemeContext.Provider value={contextValue}>{children}</ThemeContext.Provider>;
};

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (context === null) {
    throw new Error('useTheme must be used within ThemeProvider');
  }

  return context;
};
