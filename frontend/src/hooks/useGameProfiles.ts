import { useCallback, useEffect, useState } from 'react';

import {
  ActivateProfile,
  ClearActiveProfile,
  CreateProfile,
  DeleteProfile,
  ListProfiles,
  RenameProfile,
} from '@bindings/github.com/phergul/mod-manager/internal/services/profileservice';
import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage } from '@utils';

type ProfileAction = 'activate' | 'clear-active' | 'create' | 'delete' | 'rename';

export const useGameProfiles = (gameID: number) => {
  const { addToast } = useToast();
  const [profiles, setProfiles] = useState<ModProfile[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<ProfileAction | null>(null);

  const loadProfiles = useCallback(async () => {
    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedProfiles = await ListProfiles(gameID);
      setProfiles(loadedProfiles);
      return loadedProfiles;
    } catch (error) {
      const message = getErrorMessage(error);
      setLoadError(message);
      throw error;
    } finally {
      setIsLoading(false);
    }
  }, [gameID]);

  const runProfileAction = useCallback(
    async <T,>(action: ProfileAction, operation: () => Promise<T>, successMessage: string) => {
      setPendingAction(action);

      try {
        const result = await operation();
        await loadProfiles();
        addToast({
          message: successMessage,
          tone: 'success',
        });
        return result;
      } catch (error) {
        addToast({
          message: getErrorMessage(error),
          tone: 'error',
        });
        throw error;
      } finally {
        setPendingAction(null);
      }
    },
    [addToast, loadProfiles],
  );

  const createProfile = useCallback(
    (name: string) =>
      runProfileAction('create', () => CreateProfile(gameID, name), 'Profile created.'),
    [gameID, runProfileAction],
  );

  const renameProfile = useCallback(
    (profileID: number, name: string) =>
      runProfileAction('rename', () => RenameProfile(profileID, name), 'Profile renamed.'),
    [runProfileAction],
  );

  const deleteProfile = useCallback(
    (profileID: number) =>
      runProfileAction('delete', () => DeleteProfile(profileID), 'Profile deleted.'),
    [runProfileAction],
  );

  const activateProfile = useCallback(
    (profileID: number) =>
      runProfileAction(
        'activate',
        () => ActivateProfile(gameID, profileID),
        'Active profile updated.',
      ),
    [gameID, runProfileAction],
  );

  const clearActiveProfile = useCallback(
    () =>
      runProfileAction(
        'clear-active',
        () => ClearActiveProfile(gameID),
        'Active profile cleared.',
      ),
    [gameID, runProfileAction],
  );

  useEffect(() => {
    let isMounted = true;

    const loadInitialProfiles = async () => {
      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedProfiles = await ListProfiles(gameID);
        if (isMounted) {
          setProfiles(loadedProfiles);
        }
      } catch (error) {
        if (isMounted) {
          setLoadError(getErrorMessage(error));
        }
      } finally {
        if (isMounted) {
          setIsLoading(false);
        }
      }
    };

    loadInitialProfiles();

    return () => {
      isMounted = false;
    };
  }, [gameID]);

  return {
    activeProfile: profiles.find((profile) => profile.IsActive) ?? null,
    activateProfile,
    clearActiveProfile,
    createProfile,
    deleteProfile,
    isLoading,
    loadError,
    pendingAction,
    profiles,
    refreshProfiles: loadProfiles,
    renameProfile,
  };
};
