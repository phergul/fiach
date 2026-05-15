import { SettingsPage } from '@components/Settings/Common/SettingsPage/SettingsPage';
import { ModsSettings } from '@components/Settings/Mods/ModsSettings/ModsSettings';

import './Settings.scss';

export const Settings = () => {
  return (
    <SettingsPage title="Settings">
      <ModsSettings />
    </SettingsPage>
  );
};
