import { useCallback, useState } from 'react';

import {
  AddModToProfile,
  CreateProfile,
  DuplicateProfile,
  DeleteProfile,
  ListProfileMods,
  RemoveModFromProfile,
  RenameProfile,
  ReorderProfileMods,
  SetProfileModEnabled,
} from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type { ProfileMod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { invalidateDeploymentPreview } from '../../deployment/preview/deploymentPreviewResource';

import {
  fetchGameProfiles,
  gameProfilesResource,
  invalidateGameProfiles,
  preloadGameProfiles,
} from './gameProfilesResource';

export { fetchGameProfiles, gameProfilesResource, invalidateGameProfiles, preloadGameProfiles };
export type { CachedGameProfiles } from './gameProfilesResource';

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

const profileModToggleKey = (profileID: number, modID: number) => `${profileID}:${modID}`;

export const useGameProfiles = (gameID: number | null) => {
  const { addErrorToast, addToast } = useToast();
  const {
    data: gameProfiles,
    isInitialLoading,
    isLoading,
    isRefreshing,
    loadError,
    refresh,
    setCached: setGameProfiles,
  } = gameProfilesResource.useCached(gameID);
  const profiles = gameProfiles.profiles;
  const profileModsByProfileID = gameProfiles.profileModsByProfileID;
  const [pendingAction, setPendingAction] = useState<ProfileAction | null>(null);
  const [pendingProfileModToggleIDs, setPendingProfileModToggleIDs] = useState<
    Record<string, boolean>
  >({});

  const invalidateProfileDeploymentPreview = useCallback((profileID: number) => {
    invalidateDeploymentPreview(profileID);
  }, []);

  const updateProfileMods = useCallback(
    (profileID: number, updater: (profileMods: ProfileMod[]) => ProfileMod[]) => {
      setGameProfiles({
        profiles: gameProfiles.profiles,
        profileModsByProfileID: {
          ...gameProfiles.profileModsByProfileID,
          [profileID]: updater(gameProfiles.profileModsByProfileID[profileID] ?? []),
        },
      });
    },
    [gameProfiles.profileModsByProfileID, gameProfiles.profiles, setGameProfiles],
  );

  const loadProfileMods = useCallback(
    async (profileID: number) => {
      const loadedProfileMods = await ListProfileMods(profileID);
      setGameProfiles({
        profiles: gameProfiles.profiles,
        profileModsByProfileID: {
          ...gameProfiles.profileModsByProfileID,
          [profileID]: loadedProfileMods,
        },
      });
      invalidateProfileDeploymentPreview(profileID);
      return loadedProfileMods;
    },
    [
      gameProfiles.profileModsByProfileID,
      gameProfiles.profiles,
      invalidateProfileDeploymentPreview,
      setGameProfiles,
    ],
  );

  const refreshProfiles = useCallback(() => refresh(), [refresh]);

  const runProfileAction = useCallback(
    async <T>(action: ProfileAction, operation: () => Promise<T>, successMessage: string) => {
      setPendingAction(action);

      try {
        const result = await operation();
        await refresh();
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
    [addErrorToast, addToast, refresh],
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
      runProfileAction(
        'delete',
        async () => {
          await DeleteProfile(profileID);
          invalidateProfileDeploymentPreview(profileID);
        },
        'Profile deleted.',
      ),
    [invalidateProfileDeploymentPreview, runProfileAction],
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
    async (profileID: number, modID: number, enabled: boolean) => {
      const pendingKey = profileModToggleKey(profileID, modID);
      const previousProfileMods = profileModsByProfileID[profileID] ?? [];

      setPendingProfileModToggleIDs((currentPendingIDs) => ({
        ...currentPendingIDs,
        [pendingKey]: true,
      }));
      updateProfileMods(profileID, (currentProfileMods) =>
        currentProfileMods.map((profileMod) =>
          profileMod.ModID === modID ? { ...profileMod, Enabled: enabled } : profileMod,
        ),
      );

      try {
        const updatedProfileMod = await SetProfileModEnabled(profileID, modID, enabled);
        updateProfileMods(profileID, (currentProfileMods) =>
          currentProfileMods.map((profileMod) =>
            profileMod.ModID === modID ? updatedProfileMod : profileMod,
          ),
        );
        invalidateProfileDeploymentPreview(profileID);
        addToast({
          message: enabled ? 'Mod enabled for profile.' : 'Mod disabled for profile.',
          tone: 'success',
        });
        return updatedProfileMod;
      } catch (error) {
        updateProfileMods(profileID, () => previousProfileMods);
        addErrorToast(error);
        throw error;
      } finally {
        setPendingProfileModToggleIDs((currentPendingIDs) => {
          const nextPendingIDs = { ...currentPendingIDs };
          delete nextPendingIDs[pendingKey];
          return nextPendingIDs;
        });
      }
    },
    [
      addErrorToast,
      addToast,
      invalidateProfileDeploymentPreview,
      profileModsByProfileID,
      updateProfileMods,
    ],
  );

  const reorderProfileMods = useCallback(
    async (profileID: number, orderedModIDs: number[]) => {
      setPendingAction('reorder-mods');

      try {
        const reorderedProfileMods = await ReorderProfileMods(profileID, orderedModIDs);
        setGameProfiles({
          profiles: gameProfiles.profiles,
          profileModsByProfileID: {
            ...gameProfiles.profileModsByProfileID,
            [profileID]: reorderedProfileMods,
          },
        });
        invalidateProfileDeploymentPreview(profileID);
        return reorderedProfileMods;
      } catch (error) {
        addErrorToast(error);
        throw error;
      } finally {
        setPendingAction(null);
      }
    },
    [
      addErrorToast,
      gameProfiles.profileModsByProfileID,
      gameProfiles.profiles,
      invalidateProfileDeploymentPreview,
      setGameProfiles,
    ],
  );

  return {
    addModToProfile,
    addModsToProfile,
    createProfile,
    duplicateProfile,
    deleteProfile,
    isLoading,
    loadError,
    pendingAction,
    pendingProfileModToggleIDs,
    profileModsByProfileID,
    profiles,
    isInitialLoading,
    isRefreshing,
    removeModFromProfile,
    reorderProfileMods,
    refreshProfiles,
    renameProfile,
    setProfileModEnabled,
  };
};

export type UseGameProfilesResult = ReturnType<typeof useGameProfiles>;
