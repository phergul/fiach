import {
  ListProfileMods,
  ListProfiles,
} from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type {
  ModProfile,
  ProfileMod,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { createKeyedCachedResource } from '../../cache/createKeyedCachedResource';

export interface CachedGameProfiles {
  profileModsByProfileID: Record<number, ProfileMod[]>;
  profiles: ModProfile[];
}

const emptyGameProfiles: CachedGameProfiles = {
  profileModsByProfileID: {},
  profiles: [],
};

const loadProfileModEntries = async (profiles: ModProfile[]) => {
  return Promise.all(
    profiles.map(async (profile) => {
      const loadedProfileMods = await ListProfileMods(profile.ID);
      return [profile.ID, loadedProfileMods] as const;
    }),
  );
};

export const gameProfilesResource = createKeyedCachedResource<CachedGameProfiles, number>({
  emptyValue: emptyGameProfiles,
  fetch: async (gameID) => {
    const loadedProfiles = await ListProfiles(gameID);
    const profileModEntries = await loadProfileModEntries(loadedProfiles);
    return {
      profileModsByProfileID: Object.fromEntries(profileModEntries),
      profiles: loadedProfiles,
    };
  },
  isEmpty: (data) => data.profiles.length === 0,
});

export const fetchGameProfiles = gameProfilesResource.fetch;
export const preloadGameProfiles = gameProfilesResource.preload;
export const invalidateGameProfiles = gameProfilesResource.invalidate;
