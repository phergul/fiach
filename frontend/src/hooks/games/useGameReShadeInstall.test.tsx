import { act, renderHook } from '@testing-library/react';
import { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';

import { ToastProvider } from '@components/Common/Toast/Toast';

import { useGameReShadeInstall } from './useGameReShadeInstall';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/reshadeservice', () => ({
  DownloadAndOpenReShadeAddonInstaller: vi.fn(),
  DownloadAndOpenReShadeInstaller: vi.fn().mockResolvedValue({ Version: '6.0' }),
}));

vi.mock('@bindings/github.com/phergul/fiach/internal/services/windowservice', () => ({
  OpenLogsWindow: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => <ToastProvider>{children}</ToastProvider>;

describe('useGameReShadeInstall', () => {
  it('waits for explicit completion before refreshing detection', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useGameReShadeInstall({
      game: { ID: 1 } as never,
      onMenuClose: vi.fn(),
      reShadeDetection: {
        isLoading: false,
        loadError: null,
        refresh,
        result: { Status: 'not_installed' },
      } as never,
    }), { wrapper });

    await act(async () => {
      await result.current.downloadAndOpenInstaller();
    });
    expect(result.current.isCompletionPromptOpen).toBe(true);
    expect(refresh).not.toHaveBeenCalled();

    await act(async () => {
      await result.current.confirmInstallerFinished();
    });
    expect(refresh).toHaveBeenCalledOnce();
    expect(result.current.isCompletionPromptOpen).toBe(false);
  });
});
