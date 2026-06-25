import { useCallback, useEffect, useState } from 'react';

import {
  AddModToProfile,
  CreateProfile,
  DuplicateProfile,
  DeleteProfile,
  ListProfileMods,
  ListProfiles,
  RemoveModFromProfile,
  RenameProfile,
  ReorderProfileMods,
  SetProfileModEnabled,
} from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type {
  ModProfile,
  ProfileMod,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { getErrorMessage } from '@utils';

type ProfileAction =
  | 'add-mod'
  | 'add-mods'
  | 'create'
  | 'duplicate'
  | 'delete'
  | 'remove-mod'
  | 'reorder-mods'
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
  const { addErrorToast, addToast } = useToast();
  const [profiles, setProfiles] = useState<ModProfile[]>([]);
  const [profileModsByProfileID, setProfileModsByProfileID] = useState<
    Record<number, ProfileMod[]>
  >({});
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

  const loadProfiles = useCallback(
    async (isCurrent: () => boolean = () => true) => {
      if (gameID === null) {
        if (isCurrent()) {
          setProfiles([]);
          setProfileModsByProfileID({});
          setIsLoading(false);
          setLoadError(null);
        }
        return [];
      }

      if (isCurrent()) {
        setIsLoading(true);
        setLoadError(null);
      }

      try {
        const loadedProfiles = await ListProfiles(gameID);
        const profileModEntries = await loadProfileModEntries(loadedProfiles);
        if (isCurrent()) {
          setProfiles(loadedProfiles);
          setProfileModsByProfileID(Object.fromEntries(profileModEntries));
        }
        return loadedProfiles;
      } catch (error) {
        const message = getErrorMessage(error);
        if (isCurrent()) {
          setLoadError(message);
        }
        throw error;
      } finally {
        if (isCurrent()) {
          setIsLoading(false);
        }
      }
    },
    [gameID],
  );

  const refreshProfiles = useCallback(() => loadProfiles(), [loadProfiles]);

  const runProfileAction = useCallback(
    async <T>(action: ProfileAction, operation: () => Promise<T>, successMessage: string) => {
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
        addErrorToast(error);
        throw error;
      } finally {
        setPendingAction(null);
      }
    },
    [addErrorToast, addToast, loadProfiles],
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

  const duplicateProfile = useCallback(
    (profileID: number) => {
      if (gameID === null) {
        return Promise.reject(new Error('game is not selected'));
      }

      return runProfileAction(
        'duplicate',
        () => DuplicateProfile(profileID),
        'Profile duplicated.',
      );
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

  const addModsToProfile = useCallback(
    async (profileID: number, modIDs: number[]) => {
      if (modIDs.length === 0) {
        return;
      }

      setPendingAction('add-mods');

      try {
        for (const modID of modIDs) {
          await AddModToProfile(profileID, modID);
        }

        await loadProfileMods(profileID);
        addToast({
          message: `${modIDs.length} ${modIDs.length === 1 ? 'mod' : 'mods'} added to profile.`,
          tone: 'success',
        });
      } catch (error) {
        addErrorToast(error);
        throw error;
      } finally {
        setPendingAction(null);
      }
    },
    [addErrorToast, addToast, loadProfileMods],
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

  const reorderProfileMods = useCallback(
    async (profileID: number, orderedModIDs: number[]) => {
      setPendingAction('reorder-mods');

      try {
        const reorderedProfileMods = await ReorderProfileMods(profileID, orderedModIDs);
        setProfileModsByProfileID((currentProfileMods) => ({
          ...currentProfileMods,
          [profileID]: reorderedProfileMods,
        }));
        return reorderedProfileMods;
      } catch (error) {
        addErrorToast(error);
        throw error;
      } finally {
        setPendingAction(null);
      }
    },
    [addErrorToast],
  );

  useEffect(() => {
    let isMounted = true;

    loadProfiles(() => isMounted).catch(() => {
      // Load errors are stored in hook state for the caller to render.
    });

    return () => {
      isMounted = false;
    };
  }, [loadProfiles]);

  return {
    addModToProfile,
    addModsToProfile,
    createProfile,
    duplicateProfile,
    deleteProfile,
    isLoading,
    loadError,
    pendingAction,
    profileModsByProfileID,
    profiles,
    removeModFromProfile,
    reorderProfileMods,
    refreshProfiles,
    renameProfile,
    setProfileModEnabled,
  };
};

export type UseGameProfilesResult = ReturnType<typeof useGameProfiles>;
