import { useEffect, useState } from 'react';

import {
  GetGlobalModStorageRoot,
  SetGlobalModStorageRoot,
} from '@bindings/github.com/phergul/fiach/internal/services/settingsservice';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { useToast } from '@components/Common/Toast/Toast';
import { SettingsRow } from '@components/Settings/Common/SettingsRow/SettingsRow';
import { SettingsSection } from '@components/Settings/Common/SettingsSection/SettingsSection';
import { ModStorageRootPathControl } from '@components/Settings/Mods/ModStorageRootPathControl/ModStorageRootPathControl';
import { getErrorMessage, openDirectory } from '@utils';

interface PendingGlobalRootChange {
  confirmLabel: string;
  path: string;
  successMessage: string;
  title: string;
}

const futureImportsOnlyMessage =
  'Changing this setting affects future imports only. Existing imported mod folders and mod rows will not be moved.';

export const ModsSettings = () => {
  const { addErrorToast, addToast } = useToast();
  const [globalRoot, setGlobalRoot] = useState('');
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [isApplyingGlobalRoot, setIsApplyingGlobalRoot] = useState(false);
  const [pendingGlobalRootChange, setPendingGlobalRootChange] = useState<PendingGlobalRootChange | null>(null);

  useEffect(() => {
    let isMounted = true;

    const loadGlobalRoot = async () => {
      setIsLoading(true);
      setLoadError(null);

      try {
        const root = await GetGlobalModStorageRoot();
        if (isMounted) {
          setGlobalRoot(root);
        }
      } catch (error) {
        const message = getErrorMessage(error);
        if (isMounted) {
          setLoadError(message);
        }
        addErrorToast(error);
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    void loadGlobalRoot();

    return () => {
      isMounted = false;
    };
  }, [addErrorToast]);

  const requestSetGlobalRoot = async () => {
    if (isLoading || isApplyingGlobalRoot) {
      return;
    }

    try {
      const path = await openDirectory({
        buttonText: 'Use Folder',
        canCreateDirectories: true,
        title: 'Select global mod storage root',
      });
      if (path === null) {
        return;
      }

      setPendingGlobalRootChange({
        confirmLabel: 'Set Root',
        path,
        successMessage: 'Global mod storage root set.',
        title: 'Set Global Mod Storage Root?',
      });
    } catch (error) {
      addErrorToast(error);
    }
  };

  const requestClearGlobalRoot = () => {
    if (isLoading || isApplyingGlobalRoot || globalRoot.trim() === '') {
      return;
    }

    setPendingGlobalRootChange({
      confirmLabel: 'Clear Root',
      path: '',
      successMessage: 'Global mod storage root cleared.',
      title: 'Clear Global Mod Storage Root?',
    });
  };

  const applyGlobalRootChange = async () => {
    if (pendingGlobalRootChange === null || isApplyingGlobalRoot) {
      return;
    }

    setIsApplyingGlobalRoot(true);

    try {
      await SetGlobalModStorageRoot(pendingGlobalRootChange.path);
      const root = await GetGlobalModStorageRoot();
      setGlobalRoot(root);
      setLoadError(null);
      addToast({
        message: pendingGlobalRootChange.successMessage,
        tone: 'success',
      });
      setPendingGlobalRootChange(null);
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsApplyingGlobalRoot(false);
    }
  };

  return (
    <>
      <div className="mods-settings">
        <SettingsSection title="Mods Settings">
          <SettingsRow
            description={futureImportsOnlyMessage}
            status={loadError}
            title="Global Mod Storage Root"
          >
            <ModStorageRootPathControl
              isBusy={isLoading || isApplyingGlobalRoot}
              onChooseFolder={requestSetGlobalRoot}
              onClear={requestClearGlobalRoot}
              value={isLoading ? 'Loading...' : globalRoot}
            />
          </SettingsRow>
        </SettingsSection>
      </div>

      <ConfirmDialog
        confirmLabel={pendingGlobalRootChange?.confirmLabel}
        confirmTone="default"
        isBusy={isApplyingGlobalRoot}
        isOpen={pendingGlobalRootChange !== null}
        message={futureImportsOnlyMessage}
        onCancel={() => {
          if (!isApplyingGlobalRoot) {
            setPendingGlobalRootChange(null);
          }
        }}
        onConfirm={applyGlobalRootChange}
        title={pendingGlobalRootChange?.title ?? 'Confirm Storage Change'}
      />
    </>
  );
};
