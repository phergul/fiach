import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  GetAppliedProfileSummary,
  RestoreVanillaState,
} from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type {
  AppliedProfileSummary,
  RestoreResult,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ToastProvider } from '@components/Common/Toast/Toast';

import { appliedProfileResource, useAppliedProfile } from './useAppliedProfile';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/profileservice', () => ({
  GetAppliedProfileSummary: vi.fn(),
  RestoreVanillaState: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <ToastProvider>{children}</ToastProvider>
);

const appliedProfile = {
  ProfileID: 31,
  ProfileName: 'Cached Applied Profile',
} as AppliedProfileSummary;

describe('useAppliedProfile', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    appliedProfileResource.clear();
    vi.mocked(RestoreVanillaState).mockResolvedValue({
      CompletedCount: 0,
      FailedCount: 0,
      Results: [],
      SkippedCount: 0,
      Success: true,
    } as unknown as RestoreResult);
  });

  it('returns a cached applied profile immediately for the same game', async () => {
    vi.mocked(GetAppliedProfileSummary).mockResolvedValue(appliedProfile);

    const firstHook = renderHook(() => useAppliedProfile(67), { wrapper });

    await waitFor(() => {
      expect(firstHook.result.current.appliedProfile).toEqual(appliedProfile);
    });
    firstHook.unmount();

    const secondHook = renderHook(() => useAppliedProfile(67), { wrapper });

    expect(secondHook.result.current.appliedProfile).toEqual(appliedProfile);
    expect(secondHook.result.current.isInitialLoading).toBe(false);
    expect(secondHook.result.current.isRefreshing).toBe(true);
  });

  it('refetches when refreshAppliedProfile is called even with a cached game', async () => {
    const updatedAppliedProfile = {
      ProfileID: 32,
      ProfileName: 'Updated Applied Profile',
    } as AppliedProfileSummary;
    vi.mocked(GetAppliedProfileSummary)
      .mockResolvedValueOnce(appliedProfile)
      .mockResolvedValueOnce(updatedAppliedProfile);

    const { result } = renderHook(() => useAppliedProfile(68), { wrapper });

    await waitFor(() => {
      expect(result.current.appliedProfile).toEqual(appliedProfile);
    });
    expect(GetAppliedProfileSummary).toHaveBeenCalledOnce();

    await result.current.refreshAppliedProfile();

    await waitFor(() => {
      expect(result.current.appliedProfile).toEqual(updatedAppliedProfile);
    });
    expect(GetAppliedProfileSummary).toHaveBeenCalledTimes(2);
  });
});
