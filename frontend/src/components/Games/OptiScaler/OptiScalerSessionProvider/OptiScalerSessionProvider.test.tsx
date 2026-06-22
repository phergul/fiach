import { act, renderHook } from '@testing-library/react';
import { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { GetOptiScalerReleaseStatus } from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';

import { OptiScalerSessionProvider, useOptiScalerSession } from './OptiScalerSessionProvider';

vi.mock('@bindings/github.com/phergul/fiach/internal/services/optiscalerservice', () => ({
  GetOptiScalerReleaseStatus: vi.fn(),
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <OptiScalerSessionProvider>{children}</OptiScalerSessionProvider>
);

describe('OptiScalerSessionProvider', () => {
  beforeEach(() => {
    vi.mocked(GetOptiScalerReleaseStatus).mockReset();
  });

  it('loads the stable release only once per provider session', async () => {
    vi.mocked(GetOptiScalerReleaseStatus).mockResolvedValue({
      assetName: 'OptiScaler.7z',
      digest: 'digest',
      size: 1,
      tag: 'v1',
      url: 'https://example.invalid/release',
      version: 'OptiScaler v1',
    });
    const { result } = renderHook(() => useOptiScalerSession(), { wrapper });

    await act(async () => {
      await Promise.all([result.current.loadRelease(), result.current.loadRelease()]);
    });
    await act(async () => {
      await result.current.loadRelease();
    });

    expect(GetOptiScalerReleaseStatus).toHaveBeenCalledOnce();
    expect(GetOptiScalerReleaseStatus).toHaveBeenCalledWith(false);
  });

  it('retries after the release status returns an error', async () => {
    vi.mocked(GetOptiScalerReleaseStatus)
      .mockResolvedValueOnce({
        assetName: '',
        digest: '',
        error: 'GitHub returned 403 Forbidden',
        size: 0,
        tag: '',
        url: '',
        version: '',
      })
      .mockResolvedValueOnce({
        assetName: 'OptiScaler.7z',
        digest: 'digest',
        size: 1,
        tag: 'v1',
        url: 'https://example.invalid/release',
        version: 'OptiScaler v1',
      });
    const { result } = renderHook(() => useOptiScalerSession(), { wrapper });

    await act(async () => {
      await result.current.loadRelease();
    });
    await act(async () => {
      await result.current.loadRelease();
    });

    expect(GetOptiScalerReleaseStatus).toHaveBeenCalledTimes(2);
    expect(GetOptiScalerReleaseStatus).toHaveBeenNthCalledWith(1, false);
    expect(GetOptiScalerReleaseStatus).toHaveBeenNthCalledWith(2, false);
    expect(result.current.release?.version).toBe('OptiScaler v1');
    expect(result.current.releaseError).toBeNull();
  });

  it('retries after the release status call rejects', async () => {
    vi.mocked(GetOptiScalerReleaseStatus)
      .mockRejectedValueOnce(new Error('network unavailable'))
      .mockResolvedValueOnce({
        assetName: 'OptiScaler.7z',
        digest: 'digest',
        size: 1,
        tag: 'v1',
        url: 'https://example.invalid/release',
        version: 'OptiScaler v1',
      });
    const { result } = renderHook(() => useOptiScalerSession(), { wrapper });

    await act(async () => {
      await result.current.loadRelease();
    });
    await act(async () => {
      await result.current.loadRelease();
    });

    expect(GetOptiScalerReleaseStatus).toHaveBeenCalledTimes(2);
    expect(GetOptiScalerReleaseStatus).toHaveBeenNthCalledWith(1, false);
    expect(GetOptiScalerReleaseStatus).toHaveBeenNthCalledWith(2, false);
    expect(result.current.release?.version).toBe('OptiScaler v1');
    expect(result.current.releaseError).toBeNull();
  });

  it('refreshes the stable release when requested', async () => {
    vi.mocked(GetOptiScalerReleaseStatus)
      .mockResolvedValueOnce({
        assetName: 'OptiScaler.7z',
        digest: 'digest',
        size: 1,
        tag: 'v1',
        url: 'https://example.invalid/release',
        version: 'OptiScaler v1',
      })
      .mockResolvedValueOnce({
        assetName: 'OptiScaler.7z',
        digest: 'digest-2',
        size: 2,
        tag: 'v2',
        url: 'https://example.invalid/release-2',
        version: 'OptiScaler v2',
      });
    const { result } = renderHook(() => useOptiScalerSession(), { wrapper });

    await act(async () => {
      await result.current.loadRelease();
    });
    await act(async () => {
      await result.current.loadRelease(true);
    });

    expect(GetOptiScalerReleaseStatus).toHaveBeenCalledTimes(2);
    expect(GetOptiScalerReleaseStatus).toHaveBeenNthCalledWith(1, false);
    expect(GetOptiScalerReleaseStatus).toHaveBeenNthCalledWith(2, true);
    expect(result.current.release?.version).toBe('OptiScaler v2');
  });
});
