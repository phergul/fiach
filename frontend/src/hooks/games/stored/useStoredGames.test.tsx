import { renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  GetStoredGames,
  ScanAndSaveGames,
} from '@bindings/github.com/phergul/fiach/internal/services/gamesservice';
import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ToastProvider } from '@components/Common/Toast/Toast';

import { storedGamesResource, useStoredGames } from './useStoredGames';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/gamesservice', () => ({
  GetStoredGames: vi.fn(),
  ScanAndSaveGames: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <ToastProvider>{children}</ToastProvider>
);

const game = { ID: 19, Name: 'Cached Game' } as StoredGame;

describe('useStoredGames', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    storedGamesResource.clear();
    vi.mocked(ScanAndSaveGames).mockResolvedValue({
      Games: [],
      Inserted: 0,
      MarkedUnavailable: 0,
      Updated: 0,
    });
  });

  it('returns cached games immediately on a later mount', async () => {
    vi.mocked(GetStoredGames).mockResolvedValue([game]);

    const firstHook = renderHook(() => useStoredGames(), { wrapper });

    await waitFor(() => {
      expect(firstHook.result.current.games).toEqual([game]);
    });
    firstHook.unmount();

    const secondHook = renderHook(() => useStoredGames(), { wrapper });

    expect(secondHook.result.current.games).toEqual([game]);
    expect(secondHook.result.current.isInitialLoading).toBe(false);
    expect(secondHook.result.current.isRefreshing).toBe(true);
  });

  it('refetches games when retryLoadGames is called even with a cached list', async () => {
    const updatedGame = { ID: 19, Name: 'Updated Game' } as StoredGame;
    vi.mocked(GetStoredGames).mockResolvedValueOnce([game]).mockResolvedValueOnce([updatedGame]);

    const { result } = renderHook(() => useStoredGames(), { wrapper });

    await waitFor(() => {
      expect(result.current.games).toEqual([game]);
    });
    expect(GetStoredGames).toHaveBeenCalledOnce();

    await result.current.retryLoadGames();

    await waitFor(() => {
      expect(result.current.games).toEqual([updatedGame]);
    });
    expect(GetStoredGames).toHaveBeenCalledTimes(2);
  });
});
