import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  AddModToProfile,
  CreateProfile,
  DeleteProfile,
  DuplicateProfile,
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
import { ToastProvider } from '@components/Common/Toast/Toast';

import { gameProfilesResource, useGameProfiles } from './useGameProfiles';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/profileservice', () => ({
  AddModToProfile: vi.fn(),
  CreateProfile: vi.fn(),
  DeleteProfile: vi.fn(),
  DuplicateProfile: vi.fn(),
  ListProfileMods: vi.fn(),
  ListProfiles: vi.fn(),
  RemoveModFromProfile: vi.fn(),
  RenameProfile: vi.fn(),
  ReorderProfileMods: vi.fn(),
  SetProfileModEnabled: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <ToastProvider>{children}</ToastProvider>
);

const profile = { ID: 23, Name: 'Cached Profile' } as ModProfile;
const profileMod = { Enabled: true, ModID: 7, ProfileID: 23 } as ProfileMod;

describe('useGameProfiles', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    gameProfilesResource.clear();
    vi.mocked(AddModToProfile).mockResolvedValue(profileMod);
    vi.mocked(CreateProfile).mockResolvedValue(profile);
    vi.mocked(DeleteProfile).mockResolvedValue(undefined);
    vi.mocked(DuplicateProfile).mockResolvedValue(profile);
    vi.mocked(RemoveModFromProfile).mockResolvedValue(undefined);
    vi.mocked(RenameProfile).mockResolvedValue(profile);
    vi.mocked(ReorderProfileMods).mockResolvedValue([profileMod]);
    vi.mocked(SetProfileModEnabled).mockResolvedValue(profileMod);
  });

  it('returns cached profiles immediately for the same game', async () => {
    vi.mocked(ListProfiles).mockResolvedValue([profile]);
    vi.mocked(ListProfileMods).mockResolvedValue([profileMod]);

    const firstHook = renderHook(() => useGameProfiles(91), { wrapper });

    await waitFor(() => {
      expect(firstHook.result.current.profiles).toEqual([profile]);
    });
    firstHook.unmount();

    const secondHook = renderHook(() => useGameProfiles(91), { wrapper });

    expect(secondHook.result.current.profiles).toEqual([profile]);
    expect(secondHook.result.current.profileModsByProfileID[23]).toEqual([profileMod]);
    expect(secondHook.result.current.isInitialLoading).toBe(false);
    expect(secondHook.result.current.isRefreshing).toBe(true);
  });

  it('optimistically updates a profile mod toggle without reloading all profiles', async () => {
    vi.mocked(ListProfiles).mockResolvedValue([profile]);
    vi.mocked(ListProfileMods).mockResolvedValue([profileMod]);
    let resolveToggle: (profileMod: ProfileMod) => void = () => undefined;
    const pendingToggle = new Promise<ProfileMod>((resolve) => {
      resolveToggle = resolve;
    }) as ReturnType<typeof SetProfileModEnabled>;
    vi.mocked(SetProfileModEnabled).mockReturnValue(pendingToggle);

    const { result } = renderHook(() => useGameProfiles(92), { wrapper });

    await waitFor(() => {
      expect(result.current.profileModsByProfileID[23]).toEqual([profileMod]);
    });

    const togglePromise = result.current.setProfileModEnabled(23, 7, false);

    await waitFor(() => {
      expect(result.current.profileModsByProfileID[23][0].Enabled).toBe(false);
    });
    expect(result.current.pendingProfileModToggleIDs['23:7']).toBe(true);
    expect(ListProfiles).toHaveBeenCalledOnce();

    resolveToggle({ ...profileMod, Enabled: false });
    await togglePromise;

    await waitFor(() => {
      expect(result.current.pendingProfileModToggleIDs['23:7']).toBeUndefined();
    });
    expect(ListProfiles).toHaveBeenCalledOnce();
  });

  it('refetches profiles when refreshProfiles is called even with a cached game', async () => {
    const updatedProfile = { ID: 23, Name: 'Updated Profile' } as ModProfile;
    vi.mocked(ListProfiles)
      .mockResolvedValueOnce([profile])
      .mockResolvedValueOnce([updatedProfile]);
    vi.mocked(ListProfileMods).mockResolvedValue([profileMod]);

    const { result } = renderHook(() => useGameProfiles(93), { wrapper });

    await waitFor(() => {
      expect(result.current.profiles).toEqual([profile]);
    });
    expect(ListProfiles).toHaveBeenCalledOnce();

    await result.current.refreshProfiles();

    await waitFor(() => {
      expect(result.current.profiles).toEqual([updatedProfile]);
    });
    expect(ListProfiles).toHaveBeenCalledTimes(2);
  });
});
