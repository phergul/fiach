import { useCallback, useEffect, useMemo, useState } from 'react';

import type {
  ReShadeChainTarget,
  ReShadeContentCatalogue,
  ReShadeDiscoveryResult,
  ReShadeInstallerStatus,
  ReShadeRecoveryState,
  ReShadeTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  DiscoverReShadeCandidates,
  GetReShadeInstallerStatus,
  GetReShadeRecoveryState,
  ListReShadeChainTargets,
  ListReShadeContentCatalogue,
  ListReShadeTargets,
  RollbackReShadeRecovery,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import { getErrorMessage } from '@utils';

import { useRuntime } from '../../runtime/useRuntime';

export type ReShadeAggregateStatus =
  | 'checking'
  | 'error'
  | 'recovery'
  | 'conflict'
  | 'drift'
  | 'update'
  | 'managed'
  | 'unmanaged'
  | 'not_detected';

interface CachedGameReShade {
  catalogue: ReShadeContentCatalogue | null;
  chainTargets: ReShadeChainTarget[];
  discovery: ReShadeDiscoveryResult | null;
  installerStatus: ReShadeInstallerStatus | null;
  recovery: ReShadeRecoveryState | null;
  targets: ReShadeTarget[];
}

const cachedGameReShadeByGameID = new Map<number, CachedGameReShade>();

const releaseForTarget = (
  target: ReShadeTarget,
  installerStatus: ReShadeInstallerStatus | null,
) => {
  if (installerStatus === null) {
    return null;
  }
  return target.BuildVariant === 'addon' ? installerStatus.addon : installerStatus.standard;
};

const hasDetectedUnmanagedReShade = (discovery: ReShadeDiscoveryResult | null) =>
  discovery?.candidates.some((candidate) =>
    candidate.proxyEvidence.some((evidence) => evidence.isReShade),
  ) ?? false;

export const isReShadeUpdateAvailable = (
  target: ReShadeTarget,
  installerStatus: ReShadeInstallerStatus | null,
) => {
  const release = releaseForTarget(target, installerStatus);
  return (
    release !== null &&
    release.error === '' &&
    release.version !== '' &&
    target.RuntimeVersion !== '' &&
    target.RuntimeVersion !== release.version
  );
};

export const getReShadeAggregateStatus = (
  discovery: ReShadeDiscoveryResult | null,
  targets: ReShadeTarget[],
  recovery: ReShadeRecoveryState | null,
  installerStatus: ReShadeInstallerStatus | null,
): ReShadeAggregateStatus => {
  if (recovery?.required) {
    return 'recovery';
  }
  if (targets.some((target) => target.Status === 'recovery_required')) {
    return 'recovery';
  }
  if (discovery?.candidates.some((candidate) => candidate.conflicts.length > 0)) {
    return 'conflict';
  }
  if (
    targets.some(
      (target) => target.Status === 'drifted' || target.Status === 'incompatible_manifest',
    )
  ) {
    return 'drift';
  }
  if (targets.some((target) => isReShadeUpdateAvailable(target, installerStatus))) {
    return 'update';
  }
  if (targets.length > 0) {
    return 'managed';
  }
  if (hasDetectedUnmanagedReShade(discovery)) {
    return 'unmanaged';
  }
  return 'not_detected';
};

export const useGameReShade = (gameID: number | null) => {
  const { isWindows } = useRuntime();
  const cachedReShade = gameID === null ? undefined : cachedGameReShadeByGameID.get(gameID);
  const [discovery, setDiscovery] = useState<ReShadeDiscoveryResult | null>(
    cachedReShade?.discovery ?? null,
  );
  const [targets, setTargets] = useState<ReShadeTarget[]>(cachedReShade?.targets ?? []);
  const [recovery, setRecovery] = useState<ReShadeRecoveryState | null>(
    cachedReShade?.recovery ?? null,
  );
  const [installerStatus, setInstallerStatus] = useState<ReShadeInstallerStatus | null>(
    cachedReShade?.installerStatus ?? null,
  );
  const [catalogue, setCatalogue] = useState<ReShadeContentCatalogue | null>(
    cachedReShade?.catalogue ?? null,
  );
  const [chainTargets, setChainTargets] = useState<ReShadeChainTarget[]>(
    cachedReShade?.chainTargets ?? [],
  );
  const [isLoading, setIsLoading] = useState(gameID !== null && cachedReShade === undefined);
  const [isRollingBack, setIsRollingBack] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const refresh = useCallback(
    async (refreshRemote = false) => {
      if (gameID === null || !isWindows) {
        setDiscovery(null);
        setTargets([]);
        setRecovery(null);
        setInstallerStatus(null);
        setCatalogue(null);
        setChainTargets([]);
        setLoadError(null);
        setIsLoading(false);
        return;
      }

      const cachedCurrentReShade = cachedGameReShadeByGameID.get(gameID);
      if (cachedCurrentReShade === undefined) {
        setDiscovery(null);
        setTargets([]);
        setRecovery(null);
        setInstallerStatus(null);
        setCatalogue(null);
        setChainTargets([]);
      } else {
        setDiscovery(cachedCurrentReShade.discovery);
        setTargets(cachedCurrentReShade.targets);
        setRecovery(cachedCurrentReShade.recovery);
        setInstallerStatus(cachedCurrentReShade.installerStatus);
        setCatalogue(cachedCurrentReShade.catalogue);
        setChainTargets(cachedCurrentReShade.chainTargets);
      }
      setIsLoading(true);
      setLoadError(null);
      try {
        const [
          loadedDiscovery,
          loadedTargets,
          loadedRecovery,
          loadedInstallerStatus,
          loadedCatalogue,
          loadedChainTargets,
        ] = await Promise.all([
          DiscoverReShadeCandidates(gameID),
          ListReShadeTargets(gameID),
          GetReShadeRecoveryState(),
          GetReShadeInstallerStatus(refreshRemote),
          ListReShadeContentCatalogue(refreshRemote),
          ListReShadeChainTargets(gameID),
        ]);
        cachedGameReShadeByGameID.set(gameID, {
          catalogue: loadedCatalogue,
          chainTargets: loadedChainTargets,
          discovery: loadedDiscovery,
          installerStatus: loadedInstallerStatus,
          recovery: loadedRecovery,
          targets: loadedTargets,
        });
        setDiscovery(loadedDiscovery);
        setTargets(loadedTargets);
        setRecovery(loadedRecovery);
        setInstallerStatus(loadedInstallerStatus);
        setCatalogue(loadedCatalogue);
        setChainTargets(loadedChainTargets);
      } catch (error) {
        setLoadError(getErrorMessage(error));
      } finally {
        setIsLoading(false);
      }
    },
    [gameID, isWindows],
  );

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const rollbackRecovery = useCallback(async () => {
    if (!isWindows || !recovery?.required || recovery.journalId === undefined || isRollingBack) {
      return null;
    }
    setIsRollingBack(true);
    try {
      const result = await RollbackReShadeRecovery(recovery.journalId);
      await refresh();
      return result;
    } finally {
      setIsRollingBack(false);
    }
  }, [isRollingBack, recovery, refresh]);

  const aggregateStatus = useMemo(() => {
    if (isLoading) {
      return 'checking';
    }
    if (loadError !== null) {
      return 'error';
    }
    return getReShadeAggregateStatus(discovery, targets, recovery, installerStatus);
  }, [discovery, installerStatus, isLoading, loadError, recovery, targets]);

  return {
    aggregateStatus,
    catalogue,
    chainTargets,
    discovery,
    installerStatus,
    isInitialLoading: isLoading && discovery === null && targets.length === 0,
    isLoading,
    isRefreshing: isLoading && (discovery !== null || targets.length > 0),
    isRollingBack,
    loadError,
    recovery,
    refresh,
    rollbackRecovery,
    targets,
  };
};

export type UseGameReShadeResult = ReturnType<typeof useGameReShade>;
