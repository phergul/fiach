import { useCallback, useEffect, useState } from 'react';

import {
  ActivateProfile,
  AddModToProfile,
  CreateProfile,
  DeactivateProfile,
  DeleteProfile,
  ListProfileMods,
  ListProfiles,
  RemoveModFromProfile,
  RenameProfile,
  SetProfileModEnabled,
} from '@bindings/github.com/phergul/mod-manager/internal/services/profileservice';
import type { ModProfile, ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage } from '@utils';

type ProfileAction =
  | 'activate'
  | 'add-mod'
  | 'create'
  | 'deactivate'
  | 'delete'
  | 'remove-mod'
  | 'rename'
  | 'toggle-mod';

const loadProfileModEntries = async (profiles: ModProfile[]) => {
  return Promise.all(
    profiles.map(async (profile) => {
      const loadedProfileMods = await ListProfileMods(profile.ID);
      return [profile.ID, loadedProfileMods] as const;
    }),
  );
};

export const useGameProfiles = (gameID: number | null) => {
  const { addToast } = useToast();
  const [profiles, setProfiles] = useState<ModProfile[]>([]);
  const [profileModsByProfileID, setProfileModsByProfileID] = useState<Record<number, ProfileMod[]>>({});
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<ProfileAction | null>(null);

  const loadProfileMods = useCallback(async (profileID: number) => {
    const loadedProfileMods = await ListProfileMods(profileID);
    setProfileModsByProfileID((currentProfileMods) => ({
      ...currentProfileMods,
      [profileID]: loadedProfileMods,
    }));
    return loadedProfileMods;
  }, []);

  const loadProfiles = useCallback(async () => {
    if (gameID === null) {
      setProfiles([]);
      setProfileModsByProfileID({});
      setIsLoading(false);
      setLoadError(null);
      return [];
    }

    setIsLoading(true);
    setLoadError(null);

    try {
      const loadedProfiles = await ListProfiles(gameID);
      const profileModEntries = await loadProfileModEntries(loadedProfiles);
      setProfiles(loadedProfiles);
      setProfileModsByProfileID(Object.fromEntries(profileModEntries));
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
    (name: string) => {
      if (gameID === null) {
        return Promise.reject(new Error('game is not selected'));
      }

      return runProfileAction('create', () => CreateProfile(gameID, name), 'Profile created.');
    },
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

  const addModToProfile = useCallback(
    (profileID: number, modID: number) =>
      runProfileAction(
        'add-mod',
        async () => {
          const profileMod = await AddModToProfile(profileID, modID);
          await loadProfileMods(profileID);
          return profileMod;
        },
        'Mod added to profile.',
      ),
    [loadProfileMods, runProfileAction],
  );

  const removeModFromProfile = useCallback(
    (profileID: number, modID: number) =>
      runProfileAction(
        'remove-mod',
        async () => {
          await RemoveModFromProfile(profileID, modID);
          await loadProfileMods(profileID);
        },
        'Mod removed from profile.',
      ),
    [loadProfileMods, runProfileAction],
  );

  const setProfileModEnabled = useCallback(
    (profileID: number, modID: number, enabled: boolean) =>
      runProfileAction(
        'toggle-mod',
        async () => {
          const profileMod = await SetProfileModEnabled(profileID, modID, enabled);
          await loadProfileMods(profileID);
          return profileMod;
        },
        enabled ? 'Mod enabled for profile.' : 'Mod disabled for profile.',
      ),
    [loadProfileMods, runProfileAction],
  );

  const activateProfile = useCallback(
    (profileID: number) => {
      if (gameID === null) {
        return Promise.reject(new Error('game is not selected'));
      }

      return runProfileAction(
        'activate',
        () => ActivateProfile(gameID, profileID),
        'Active profile updated.',
      );
    },
    [gameID, runProfileAction],
  );

  const deactivateProfile = useCallback(
    () => {
      if (gameID === null) {
        return Promise.reject(new Error('game is not selected'));
      }

      return runProfileAction(
        'deactivate',
        () => DeactivateProfile(gameID),
        'Profile deactivated.',
      );
    },
    [gameID, runProfileAction],
  );

  useEffect(() => {
    let isMounted = true;

    const loadInitialProfiles = async () => {
      if (gameID === null) {
        setProfiles([]);
        setProfileModsByProfileID({});
        setLoadError(null);
        setIsLoading(false);
        return;
      }

      setIsLoading(true);
      setLoadError(null);

      try {
        const loadedProfiles = await ListProfiles(gameID);
        const profileModEntries = await loadProfileModEntries(loadedProfiles);
        if (isMounted) {
          setProfiles(loadedProfiles);
          setProfileModsByProfileID(Object.fromEntries(profileModEntries));
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
    addModToProfile,
    createProfile,
    deactivateProfile,
    deleteProfile,
    isLoading,
    loadError,
    pendingAction,
    profileModsByProfileID,
    profiles,
    removeModFromProfile,
    refreshProfiles: loadProfiles,
    renameProfile,
    setProfileModEnabled,
  };
};

export type UseGameProfilesResult = ReturnType<typeof useGameProfiles>;
