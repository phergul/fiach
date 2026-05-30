import { useState } from 'react';

import { useToast } from '@components/Common/Toast/Toast';
import { useTheme } from '@components/Common/ThemeProvider/ThemeProvider';
import { ThemeSelectControl } from '@components/Settings/Appearance/ThemeSelectControl/ThemeSelectControl';
import { SettingsRow } from '@components/Settings/Common/SettingsRow/SettingsRow';
import { SettingsSection } from '@components/Settings/Common/SettingsSection/SettingsSection';

const themeDescription =
  'Select the theme for the application.';

export const ThemeSettings = () => {
  const { addErrorToast, addToast } = useToast();
  const { activeTheme, isLoading, setTheme, themes } = useTheme();
  const [isSaving, setIsSaving] = useState(false);

  const applyTheme = async (themeID: string) => {
    if (isSaving || themeID === activeTheme.id) {
      return;
    }

    setIsSaving(true);

    try {
      await setTheme(themeID);
      addToast({
        message: 'Theme updated.',
        tone: 'success',
      });
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="theme-settings">
      <SettingsSection title="Appearance">
        <SettingsRow description={themeDescription} title="Theme">
          <ThemeSelectControl
            isBusy={isLoading || isSaving}
            onChange={(themeID) => {
              void applyTheme(themeID);
            }}
            themes={themes}
            value={activeTheme.id}
          />
        </SettingsRow>
      </SettingsSection>
    </div>
  );
};
