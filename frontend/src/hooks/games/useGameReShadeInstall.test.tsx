import { act, renderHook } from '@testing-library/react';
import { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import {
  DownloadAndOpenReShadeInstaller,
  PreflightReShadeInstaller,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import { ToastProvider } from '@components/Common/Toast/Toast';

import { useGameReShadeInstall } from './useGameReShadeInstall';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/reshadeservice', () => ({
  DownloadAndOpenReShadeAddonInstaller: vi.fn(),
  DownloadAndOpenReShadeInstaller: vi.fn().mockResolvedValue({ Version: '6.0' }),
  PreflightReShadeInstaller: vi.fn().mockResolvedValue({
    Disposition: 'ordinary',
    Message: '',
    Targets: [],
    Variant: 'standard',
  }),
}));

vi.mock('@bindings/github.com/phergul/fiach/internal/services/windowservice', () => ({
  OpenLogsWindow: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => <ToastProvider>{children}</ToastProvider>;

describe('useGameReShadeInstall', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('waits for explicit completion before refreshing detection', async () => {
    const refresh = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useGameReShadeInstall({
      game: { ID: 1 } as never,
      onCoordinate: vi.fn(),
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

  it('redirects managed DirectX targets without opening the ordinary installer', async () => {
    const onCoordinate = vi.fn();
    vi.mocked(PreflightReShadeInstaller).mockResolvedValueOnce({
      Disposition: 'coordinated',
      Message: 'Coordinate this target.',
      Targets: [{
        ExecutableRelativePath: 'Bin/Game.exe',
        ProxyFilename: 'dxgi.dll',
        TargetRelativePath: 'Bin',
      }],
      Variant: 'standard',
    } as never);
    const { result } = renderHook(() => useGameReShadeInstall({
      game: { ID: 1 } as never,
      onCoordinate,
      onMenuClose: vi.fn(),
      reShadeDetection: {
        isLoading: false,
        loadError: null,
        refresh: vi.fn(),
        result: { Status: 'installed' },
      } as never,
    }), { wrapper });

    await act(async () => {
      await result.current.downloadAndOpenInstaller();
    });

    expect(onCoordinate).toHaveBeenCalledOnce();
    expect(DownloadAndOpenReShadeInstaller).not.toHaveBeenCalled();
    expect(result.current.isCompletionPromptOpen).toBe(false);
  });
});
