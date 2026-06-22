import { useCallback, useEffect, useMemo, useState } from 'react';

import type {
  ManagedReShadeChainTarget,
  ManagedReShadeContentCatalogue,
  ManagedReShadeDiscoveryResult,
  ManagedReShadeInstallerStatus,
  ManagedReShadeRecoveryState,
  ManagedReShadeTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  DiscoverManagedReShadeCandidates,
  GetManagedReShadeInstallerStatus,
  GetManagedReShadeRecoveryState,
  ListManagedReShadeChainTargets,
  ListManagedReShadeContentCatalogue,
  ListManagedReShadeTargets,
  RollbackManagedReShadeRecovery,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import { getErrorMessage } from '@utils';

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

const releaseForTarget = (
  target: ManagedReShadeTarget,
  installerStatus: ManagedReShadeInstallerStatus | null,
) => {
  if (installerStatus === null) {
    return null;
  }
  return target.BuildVariant === 'addon' ? installerStatus.addon : installerStatus.standard;
};

export const isManagedReShadeUpdateAvailable = (
  target: ManagedReShadeTarget,
  installerStatus: ManagedReShadeInstallerStatus | null,
) => {
  const release = releaseForTarget(target, installerStatus);
  return release !== null &&
    release.error === '' &&
    release.version !== '' &&
    target.RuntimeVersion !== '' &&
    target.RuntimeVersion !== release.version;
};

export const getReShadeAggregateStatus = (
  discovery: ManagedReShadeDiscoveryResult | null,
  targets: ManagedReShadeTarget[],
  recovery: ManagedReShadeRecoveryState | null,
  installerStatus: ManagedReShadeInstallerStatus | null,
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
  if (targets.some((target) => target.Status === 'drifted' || target.Status === 'incompatible_manifest')) {
    return 'drift';
  }
  if (targets.some((target) => isManagedReShadeUpdateAvailable(target, installerStatus))) {
    return 'update';
  }
  if (targets.length > 0) {
    return 'managed';
  }
  if ((discovery?.candidates.length ?? 0) > 0) {
    return 'unmanaged';
  }
  return 'not_detected';
};

export const useGameReShade = (gameID: number | null) => {
  const [discovery, setDiscovery] = useState<ManagedReShadeDiscoveryResult | null>(null);
  const [targets, setTargets] = useState<ManagedReShadeTarget[]>([]);
  const [recovery, setRecovery] = useState<ManagedReShadeRecoveryState | null>(null);
  const [installerStatus, setInstallerStatus] = useState<ManagedReShadeInstallerStatus | null>(null);
  const [catalogue, setCatalogue] = useState<ManagedReShadeContentCatalogue | null>(null);
  const [chainTargets, setChainTargets] = useState<ManagedReShadeChainTarget[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isRollingBack, setIsRollingBack] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const refresh = useCallback(async (refreshRemote = false) => {
    if (gameID === null) {
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
        DiscoverManagedReShadeCandidates(gameID),
        ListManagedReShadeTargets(gameID),
        GetManagedReShadeRecoveryState(),
        GetManagedReShadeInstallerStatus(refreshRemote),
        ListManagedReShadeContentCatalogue(refreshRemote),
        ListManagedReShadeChainTargets(gameID),
      ]);
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
  }, [gameID]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const rollbackRecovery = useCallback(async () => {
    if (!recovery?.required || recovery.journalId === undefined || isRollingBack) {
      return null;
    }
    setIsRollingBack(true);
    try {
      const result = await RollbackManagedReShadeRecovery(recovery.journalId);
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
    isLoading,
    isRollingBack,
    loadError,
    recovery,
    refresh,
    rollbackRecovery,
    targets,
  };
};

export type UseGameReShadeResult = ReturnType<typeof useGameReShade>;
