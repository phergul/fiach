import { useEffect, useState } from 'react';

import { IsDevMode } from '@bindings/github.com/phergul/fiach/internal/services/devservice';
import {
  CheckForUpdates,
  CurrentVersion,
} from '@bindings/github.com/phergul/fiach/internal/services/settingsservice';
import { useToast } from '@components/Common/Toast/Toast';
import { SettingsRow } from '@components/Settings/Common/SettingsRow/SettingsRow';
import { SettingsSection } from '@components/Settings/Common/SettingsSection/SettingsSection';

import './UpdateSettings.scss';

const devModeDescription = 'Updates are available in production builds only.';
const updateDescription =
  'Check GitHub for a newer release. When an update is available, Fiach downloads and installs it on restart.';

export const UpdateSettings = () => {
  const { addErrorToast } = useToast();
  const [currentVersion, setCurrentVersion] = useState('');
  const [isDevMode, setIsDevMode] = useState(false);
  const [isChecking, setIsChecking] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  useEffect(() => {
    let isMounted = true;

    const load = async () => {
      try {
        const [version, devMode] = await Promise.all([CurrentVersion(), IsDevMode()]);
        if (!isMounted) {
          return;
        }

        setCurrentVersion(version);
        setIsDevMode(devMode);
        setLoadError(null);
      } catch (error) {
        if (isMounted) {
          setLoadError('Could not load version information.');
        }
        addErrorToast(error);
      }
    };

    void load();

    return () => {
      isMounted = false;
    };
  }, [addErrorToast]);

  const handleCheckForUpdates = async () => {
    if (isChecking || isDevMode) {
      return;
    }

    setIsChecking(true);

    try {
      await CheckForUpdates();
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsChecking(false);
    }
  };

  const versionLabel = currentVersion.trim() === '' ? 'Unknown' : `v${currentVersion}`;

  return (
    <div className="update-settings">
      <SettingsSection title="About">
        <SettingsRow
          description={isDevMode ? devModeDescription : updateDescription}
          status={loadError}
          title="Version"
        >
          <div className="update-settings-control">
            <p className="update-settings-version">{versionLabel}</p>
            <button
              className="update-settings-button"
              disabled={isChecking || isDevMode || loadError !== null}
              onClick={() => {
                void handleCheckForUpdates();
              }}
              type="button"
            >
              {isChecking ? 'Checking…' : 'Check for updates'}
            </button>
          </div>
        </SettingsRow>
      </SettingsSection>
    </div>
  );
};
