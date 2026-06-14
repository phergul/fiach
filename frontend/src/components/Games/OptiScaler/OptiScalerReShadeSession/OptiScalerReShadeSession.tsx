import { useEffect, useMemo, useState } from 'react';

import {
  ReShadeInstallerVariant,
  ReShadeSessionPhase,
  type ReShadeSessionState,
} from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import type { OptiScalerTarget } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ApplyOptiScalerReShadeRepair,
  CancelOptiScalerReShadeSession,
  GetOptiScalerReShadeSession,
  PreviewOptiScalerReShadeRepair,
  RescanOptiScalerReShadeSession,
  StartOptiScalerReShadeSession,
} from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';
import { DetectGameReShade } from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import { OptiScalerPreview } from '@components/Games/OptiScaler/OptiScalerPreview/OptiScalerPreview';
import { getErrorMessage } from '@utils';

import './OptiScalerReShadeSession.scss';

export interface OptiScalerReShadeRequest {
  targetRelativePath: string | null;
  variant: ReShadeInstallerVariant;
}

interface OptiScalerReShadeSessionProps {
  gameID: number;
  onActiveChange: (active: boolean) => void;
  onRefresh: () => Promise<void>;
  request: OptiScalerReShadeRequest | null;
  targets: OptiScalerTarget[];
}

type SessionPhase = 'loading' | 'idle' | 'starting' | 'rescanning' | 'applying' | 'cancelling';

const variantLabel = (variant: ReShadeInstallerVariant) =>
  variant === ReShadeInstallerVariant.ReShadeInstallerVariantAddon
    ? 'ReShade with Add-on Support'
    : 'ReShade';

export const OptiScalerReShadeSession = ({
  gameID,
  onActiveChange,
  onRefresh,
  request,
  targets,
}: OptiScalerReShadeSessionProps) => {
  const eligibleTargets = useMemo(
    () => targets.filter((target) => target.GraphicsAPI === 'directx'),
    [targets],
  );
  const [session, setSession] = useState<ReShadeSessionState | null>(null);
  const [selectedTarget, setSelectedTarget] = useState('');
  const [variant, setVariant] = useState<ReShadeInstallerVariant>(
    ReShadeInstallerVariant.ReShadeInstallerVariantStandard,
  );
  const [phase, setPhase] = useState<SessionPhase>('loading');
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [requestConsumed, setRequestConsumed] = useState(false);
  const hasRequestedSession = request !== null && !requestConsumed;
  const isActive = session !== null || hasRequestedSession;

  useEffect(() => {
    onActiveChange(isActive);
  }, [isActive, onActiveChange]);

  useEffect(() => {
    if (request === null) {
      return;
    }
    setVariant(request.variant);
    setRequestConsumed(false);
    setSelectedTarget(request.targetRelativePath ?? '');
    setMessage(null);
    setError(null);
  }, [request]);

  useEffect(() => {
    if (hasRequestedSession && selectedTarget === '' && eligibleTargets.length === 1) {
      setSelectedTarget(eligibleTargets[0].TargetRelativePath);
    }
  }, [eligibleTargets, hasRequestedSession, selectedTarget]);

  useEffect(() => {
    let cancelled = false;
    void GetOptiScalerReShadeSession()
      .then((result) => {
        if (!cancelled) {
          setSession(result);
          if (result !== null) {
            setRequestConsumed(true);
          }
        }
      })
      .catch((loadError) => {
        if (!cancelled) {
          setError(getErrorMessage(loadError));
        }
      })
      .finally(() => {
        if (!cancelled) {
          setPhase('idle');
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const refreshStates = async () => {
    await Promise.all([
      onRefresh(),
      DetectGameReShade(gameID),
    ]);
  };

  const start = async () => {
    if (selectedTarget === '' || phase !== 'idle') {
      return;
    }
    setPhase('starting');
    setError(null);
    setMessage(null);
    try {
      const nextSession = await StartOptiScalerReShadeSession({
        gameId: gameID,
        installerVariant: variant,
        targetRelativePath: selectedTarget,
      });
      setSession(nextSession);
      setRequestConsumed(true);
      setMessage(`${variantLabel(variant)} installer opened.`);
    } catch (startError) {
      setError(getErrorMessage(startError));
    } finally {
      setPhase('idle');
    }
  };

  const rescan = async () => {
    if (phase !== 'idle') {
      return;
    }
    setPhase('rescanning');
    setError(null);
    try {
      const result = await RescanOptiScalerReShadeSession();
      setSession(result.session ?? null);
      setMessage(result.message);
      await refreshStates();
    } catch (rescanError) {
      setError(getErrorMessage(rescanError));
    } finally {
      setPhase('idle');
    }
  };

  const cancel = async () => {
    if (phase !== 'idle') {
      return;
    }
    setPhase('cancelling');
    setError(null);
    try {
      const result = await CancelOptiScalerReShadeSession();
      setSession(null);
      setMessage(result.message);
      await refreshStates();
    } catch (cancelError) {
      setError(getErrorMessage(cancelError));
    } finally {
      setPhase('idle');
    }
  };

  const applyRepair = async () => {
    if (session?.preview === undefined || session.preview === null || phase !== 'idle') {
      return;
    }
    setPhase('applying');
    setError(null);
    try {
      const result = await ApplyOptiScalerReShadeRepair(session.preview.previewHash);
      setSession(null);
      setMessage(result.message);
      await refreshStates();
    } catch (applyError) {
      setError(getErrorMessage(applyError));
    } finally {
      setPhase('idle');
    }
  };

  const rebuildPreviewWithBackup = async () => {
    if (session === null || phase !== 'idle') {
      return;
    }
    setPhase('rescanning');
    setError(null);
    try {
      const preview = await PreviewOptiScalerReShadeRepair(true);
      setSession({ ...session, preview });
    } catch (previewError) {
      setError(getErrorMessage(previewError));
    } finally {
      setPhase('idle');
    }
  };

  if (phase === 'loading') {
    return null;
  }
  if (!isActive && message === null && error === null) {
    return null;
  }

  const selected = eligibleTargets.find((target) => target.TargetRelativePath === selectedTarget);
  const sessionBelongsToGame = session === null || session.gameId === gameID;

  return (
    <section className="optiscaler-reshade-session" aria-label="Coordinated ReShade session">
      <header>
        <div>
          <h2>Coordinate ReShade</h2>
          <p>Keep OptiScaler as the primary proxy while the official installer runs.</p>
        </div>
      </header>

      {!sessionBelongsToGame && (
        <p className="optiscaler-reshade-session-error">
          A coordinated ReShade session is pending for game {session?.gameId}.
        </p>
      )}

      {session === null && hasRequestedSession && (
        <div className="optiscaler-reshade-session-content">
          <label>
            Managed DirectX target
            <select value={selectedTarget} onChange={(event) => setSelectedTarget(event.target.value)}>
              <option value="">Select a target</option>
              {eligibleTargets.map((target) => (
                <option key={target.ID} value={target.TargetRelativePath}>
                  {target.ExecutableRelativePath}
                </option>
              ))}
            </select>
          </label>
          <label>
            Installer
            <select
              value={variant}
              onChange={(event) => setVariant(event.target.value as ReShadeInstallerVariant)}
            >
              <option value={ReShadeInstallerVariant.ReShadeInstallerVariantStandard}>ReShade</option>
              <option value={ReShadeInstallerVariant.ReShadeInstallerVariantAddon}>
                ReShade with Add-on Support
              </option>
            </select>
          </label>
          {selected !== undefined && (
            <p>
              In the upstream installer, select <strong>{selected.ExecutableRelativePath}</strong>.
            </p>
          )}
          <div className="optiscaler-reshade-session-actions">
            <button
              className="button-main"
              disabled={selected === undefined || phase !== 'idle'}
              onClick={() => void start()}
              type="button"
            >
              {phase === 'starting' ? 'Opening installer...' : `Open ${variantLabel(variant)} installer`}
            </button>
          </div>
        </div>
      )}

      {session !== null && sessionBelongsToGame && (
        <div className="optiscaler-reshade-session-content">
          <dl>
            <div><dt>Executable</dt><dd>{session.executableRelativePath}</dd></div>
            <div><dt>Primary proxy</dt><dd>{session.proxyFilename}</dd></div>
            <div><dt>Chained runtime</dt><dd>{session.chainedFilename}</dd></div>
            <div><dt>Installer</dt><dd>{variantLabel(session.installerVariant)}</dd></div>
          </dl>

          {session.phase === ReShadeSessionPhase.ReShadeSessionPhaseAwaitingCompletion && (
            <p>
              Select <strong>{session.executableRelativePath}</strong> in the upstream installer.
              Complete or close it before rescanning.
            </p>
          )}

          {session.phase === ReShadeSessionPhase.ReShadeSessionPhaseConflict && (
            <p className="optiscaler-reshade-session-error">
              DLL ownership is unknown: {session.conflictingPath}
            </p>
          )}

          {session.phase === ReShadeSessionPhase.ReShadeSessionPhaseRepairReady &&
            session.preview !== undefined && session.preview !== null && (
              <>
                <OptiScalerPreview preview={session.preview} />
                {session.preview.drift.length > 0 && !session.preview.request.backupAndContinue && (
                  <div className="optiscaler-reshade-session-drift">
                    <p>Drifted managed files must be archived before this repair can continue.</p>
                    <button
                      disabled={phase !== 'idle'}
                      onClick={() => void rebuildPreviewWithBackup()}
                      type="button"
                    >
                      Back up drift and rebuild preview
                    </button>
                  </div>
                )}
              </>
            )}

          <div className="optiscaler-reshade-session-actions">
            {session.phase === ReShadeSessionPhase.ReShadeSessionPhaseRepairReady ? (
              <button
                className="button-main"
                disabled={phase !== 'idle' || session.preview?.canApply !== true}
                onClick={() => void applyRepair()}
                type="button"
              >
                {phase === 'applying' ? 'Applying repair...' : 'Apply repair'}
              </button>
            ) : (
              <button
                className="button-main"
                disabled={phase !== 'idle'}
                onClick={() => void rescan()}
                type="button"
              >
                {phase === 'rescanning' ? 'Rescanning...' : 'Installer finished, rescan'}
              </button>
            )}
            <button disabled={phase !== 'idle'} onClick={() => void cancel()} type="button">
              {phase === 'cancelling' ? 'Cancelling...' : 'Cancel session'}
            </button>
          </div>
        </div>
      )}

      {message !== null && <p className="optiscaler-reshade-session-message">{message}</p>}
      {error !== null && <p className="optiscaler-reshade-session-error">{error}</p>}
    </section>
  );
};
