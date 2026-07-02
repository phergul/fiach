import { ThemeSettings } from '@components/Settings/Appearance/ThemeSettings/ThemeSettings';
import { UpdateSettings } from '@components/Settings/About/UpdateSettings/UpdateSettings';
import { SettingsPage } from '@components/Settings/Common/SettingsPage/SettingsPage';
import { ModsSettings } from '@components/Settings/Mods/ModsSettings/ModsSettings';

export const Settings = () => {
  return (
    <SettingsPage title="Settings">
      <UpdateSettings />
      <ThemeSettings />
      <ModsSettings />
    </SettingsPage>
  );
};
