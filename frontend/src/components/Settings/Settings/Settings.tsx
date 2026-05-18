import { ThemeSettings } from '@components/Settings/Appearance/ThemeSettings/ThemeSettings';
import { SettingsPage } from '@components/Settings/Common/SettingsPage/SettingsPage';
import { ModsSettings } from '@components/Settings/Mods/ModsSettings/ModsSettings';

export const Settings = () => {
  return (
    <SettingsPage title="Settings">
      <ThemeSettings />
      <ModsSettings />
    </SettingsPage>
  );
};
