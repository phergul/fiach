import { useCallback, useEffect, useMemo, useState } from 'react';

import type {
  OptiScalerCandidate,
  OptiScalerRecoveryState,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  DiscoverOptiScalerCandidates,
  GetOptiScalerRecoveryState,
  ListOptiScalerTargets,
  RollbackOptiScalerRecovery,
} from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';
import { useOptiScalerSession } from '@components/Games/OptiScaler/OptiScalerSessionProvider/OptiScalerSessionProvider';
import { getErrorMessage } from '@utils';

export type OptiScalerAggregateStatus =
  | 'checking'
  | 'error'
  | 'recovery'
  | 'drift'
  | 'update'
  | 'managed'
  | 'unmanaged'
  | 'not_detected';

export const getOptiScalerAggregateStatus = (
  candidates: OptiScalerCandidate[],
  targets: OptiScalerTarget[],
  recovery: OptiScalerRecoveryState | null,
  latestReleaseTag: string | null,
  latestReleaseDigest: string | null,
): OptiScalerAggregateStatus => {
  if (recovery?.required) {
    return 'recovery';
  }
  if (targets.some((target) => target.Status === 'recovery_required')) {
    return 'recovery';
  }
  if (targets.some((target) => target.Status === 'drifted')) {
    return 'drift';
  }
  if (
    latestReleaseTag !== null &&
    targets.some((target) =>
      (target.ReleaseTag !== '' && target.ReleaseTag !== latestReleaseTag) ||
      (latestReleaseDigest !== null && target.ReleaseDigest !== '' && target.ReleaseDigest !== latestReleaseDigest),
    )
  ) {
    return 'update';
  }
  if (targets.length > 0) {
    return 'managed';
  }
  if (candidates.some((candidate) => !candidate.managed)) {
    return 'unmanaged';
  }
  return 'not_detected';
};

export const useGameOptiScaler = (gameID: number | null) => {
  const { isReleaseLoading, loadRelease, release, releaseError } = useOptiScalerSession();
  const [candidates, setCandidates] = useState<OptiScalerCandidate[]>([]);
  const [targets, setTargets] = useState<OptiScalerTarget[]>([]);
  const [recovery, setRecovery] = useState<OptiScalerRecoveryState | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [isRollingBack, setIsRollingBack] = useState(false);

  const refresh = useCallback(async () => {
    if (gameID === null) {
      setCandidates([]);
      setTargets([]);
      setRecovery(null);
      setLoadError(null);
      setIsLoading(false);
      return;
    }

    setIsLoading(true);
    setLoadError(null);
    try {
      const [loadedCandidates, loadedTargets, loadedRecovery] = await Promise.all([
        DiscoverOptiScalerCandidates(gameID),
        ListOptiScalerTargets(gameID),
        GetOptiScalerRecoveryState(),
        loadRelease(),
      ]);
      setCandidates(loadedCandidates);
      setTargets(loadedTargets);
      setRecovery(loadedRecovery);
    } catch (error) {
      setLoadError(getErrorMessage(error));
    } finally {
      setIsLoading(false);
    }
  }, [gameID, loadRelease]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const rollbackRecovery = useCallback(async () => {
    if (!recovery?.required || recovery.journalId === undefined || isRollingBack) {
      return null;
    }
    setIsRollingBack(true);
    try {
      const result = await RollbackOptiScalerRecovery(recovery.journalId);
      await refresh();
      return result;
    } finally {
      setIsRollingBack(false);
    }
  }, [isRollingBack, recovery, refresh]);

  const aggregateStatus = useMemo(() => {
    if (isLoading || isReleaseLoading) {
      return 'checking';
    }
    if (loadError !== null) {
      return 'error';
    }
    return getOptiScalerAggregateStatus(
      candidates,
      targets,
      recovery,
      release?.tag ?? null,
      release?.digest ?? null,
    );
  }, [candidates, isLoading, isReleaseLoading, loadError, recovery, release?.digest, release?.tag, targets]);

  return {
    aggregateStatus,
    candidates,
    isLoading,
    isReleaseLoading,
    isRollingBack,
    loadError,
    recovery,
    refresh,
    release,
    releaseError,
    rollbackRecovery,
    targets,
  };
};

export type UseGameOptiScalerResult = ReturnType<typeof useGameOptiScaler>;
