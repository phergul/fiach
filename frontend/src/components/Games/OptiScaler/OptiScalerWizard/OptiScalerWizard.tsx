import { FormEvent, useEffect, useMemo, useState } from 'react';

import type {
  OptiScalerApplyResult,
  OptiScalerPreview as OptiScalerPreviewModel,
  OptiScalerRequest,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Action, GraphicsAPI } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import {
  ApplyOptiScalerAction,
  GetOptiScalerRecoveryState,
  PreviewOptiScalerAction,
} from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';
import { OptiScalerPreview } from '@components/Games/OptiScaler/OptiScalerPreview/OptiScalerPreview';
import type { OptiScalerSelection } from '@components/Games/OptiScaler/OptiScalerTargetList/OptiScalerTargetList';
import { getErrorMessage } from '@utils';

import './OptiScalerWizard.scss';

type WizardStep = 'target' | 'configuration' | 'safety' | 'preview' | 'result';
type OperationPhase = 'idle' | 'previewing' | 'applying' | 'refreshing';

interface OptiScalerWizardProps {
  gameID: number;
  onClose: () => void;
  onRecoveryRequired: () => Promise<void>;
  onRefresh: () => Promise<void>;
  selection: OptiScalerOperationSelection;
}

export interface OptiScalerOperationSelection extends OptiScalerSelection {
  action: Action;
}

const steps: WizardStep[] = ['target', 'configuration', 'safety', 'preview', 'result'];
const stepLabels: Record<WizardStep, string> = {
  target: 'Target',
  configuration: 'Configuration',
  safety: 'Safety',
  preview: 'Preview',
  result: 'Result',
};
const supportedProxyFilenames = [
  'dxgi.dll',
  'winmm.dll',
  'd3d12.dll',
  'dbghelp.dll',
  'version.dll',
  'wininet.dll',
  'winhttp.dll',
  'OptiScaler.asi',
];

const actionLabel = (action: Action) => {
  switch (action) {
    case Action.ActionInstall:
      return 'Install';
    case Action.ActionAdopt:
      return 'Adopt';
    case Action.ActionUpdate:
      return 'Update';
    case Action.ActionRepair:
      return 'Repair';
    case Action.ActionUninstall:
      return 'Uninstall';
    default:
      return 'Manage';
  }
};

export const OptiScalerWizard = ({
  gameID,
  onClose,
  onRecoveryRequired,
  onRefresh,
  selection,
}: OptiScalerWizardProps) => {
  const isNewTarget = selection.action === Action.ActionInstall || selection.action === Action.ActionAdopt;
  const isUninstall = selection.action === Action.ActionUninstall;
  const executableRelativePath =
    selection.candidate?.executableRelativePath ?? selection.target?.ExecutableRelativePath ?? '';
  const targetRelativePath =
    selection.candidate?.targetRelativePath ?? selection.target?.TargetRelativePath ?? '';
  const executableName =
    selection.candidate?.executableName ?? executableRelativePath.split(/[\\/]/).pop() ?? '';
  const [step, setStep] = useState<WizardStep>('target');
  const [graphicsAPI, setGraphicsAPI] = useState<GraphicsAPI | ''>('');
  const [proxyFilename, setProxyFilename] = useState('');
  const [dxgiSpoofing, setDXGISpoofing] = useState<boolean | null>(null);
  const [processFilter, setProcessFilter] = useState('');
  const [enableReShadeCoexistence, setEnableReShadeCoexistence] = useState(false);
  const [targetConfirmed, setTargetConfirmed] = useState(false);
  const [warningAcknowledged, setWarningAcknowledged] = useState(false);
  const [backupAndContinue, setBackupAndContinue] = useState(false);
  const [preview, setPreview] = useState<OptiScalerPreviewModel | null>(null);
  const [result, setResult] = useState<OptiScalerApplyResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [phase, setPhase] = useState<OperationPhase>('idle');
  const hasDetectedReShade = selection.candidate?.hasReShade ?? false;

  useEffect(() => {
    const storedGraphicsAPI = selection.target?.GraphicsAPI;
    const initialGraphicsAPI = storedGraphicsAPI === GraphicsAPI.GraphicsAPIVulkan
      ? GraphicsAPI.GraphicsAPIVulkan
      : storedGraphicsAPI === GraphicsAPI.GraphicsAPIDirectX
        ? GraphicsAPI.GraphicsAPIDirectX
        : '';
    setStep('target');
    setGraphicsAPI(initialGraphicsAPI);
    setProxyFilename(selection.target?.ProxyFilename ?? '');
    setDXGISpoofing(selection.target === null ? null : selection.target.DXGISpoofing);
    setProcessFilter(selection.target === null ? executableName : selection.target.ProcessFilter ?? '');
    setEnableReShadeCoexistence(selection.target?.EnableReShadeCoexistence ?? false);
    setTargetConfirmed(false);
    setWarningAcknowledged(false);
    setBackupAndContinue(false);
    setPreview(null);
    setResult(null);
    setError(null);
    setPhase('idle');
  }, [executableName, selection]);

  const request = useMemo<OptiScalerRequest | null>(() => {
    if (graphicsAPI === '' || dxgiSpoofing === null || proxyFilename === '') {
      return null;
    }
    return {
      acknowledgeWarning: isNewTarget && warningAcknowledged,
      action: selection.action,
      backupAndContinue,
      dxgiSpoofing,
      enableReShadeCoexistence,
      executableRelativePath,
      gameId: gameID,
      graphicsApi: graphicsAPI,
      processFilter: processFilter.trim() === '' ? null : processFilter.trim(),
      proxyFilename,
      targetRelativePath,
    };
  }, [
    backupAndContinue,
    dxgiSpoofing,
    enableReShadeCoexistence,
    executableRelativePath,
    gameID,
    graphicsAPI,
    isNewTarget,
    processFilter,
    proxyFilename,
    selection.action,
    targetRelativePath,
    warningAcknowledged,
  ]);

  const chooseGraphicsAPI = (nextAPI: GraphicsAPI | '') => {
    setGraphicsAPI(nextAPI);
    if (nextAPI === GraphicsAPI.GraphicsAPIDirectX) {
      setProxyFilename('dxgi.dll');
    } else if (nextAPI === GraphicsAPI.GraphicsAPIVulkan) {
      setProxyFilename('winmm.dll');
      setEnableReShadeCoexistence(false);
    } else {
      setProxyFilename('');
    }
    setPreview(null);
  };

  const loadPreview = async (nextRequest: OptiScalerRequest | null = request) => {
    if (nextRequest === null) {
      return;
    }
    setPhase('previewing');
    setError(null);
    try {
      setPreview(await PreviewOptiScalerAction(nextRequest));
      setStep('preview');
    } catch (previewError) {
      setError(getErrorMessage(previewError));
    } finally {
      setPhase('idle');
    }
  };

  const apply = async () => {
    if (request === null || preview === null || phase !== 'idle') {
      return;
    }
    setPhase('applying');
    setError(null);
    try {
      const applyResult = await ApplyOptiScalerAction(request, preview.previewHash);
      setResult(applyResult);
      setStep('result');
      setPhase('refreshing');
      await onRefresh();
      setPhase('idle');
    } catch (applyError) {
      setError(getErrorMessage(applyError));
      const recovery = await GetOptiScalerRecoveryState().catch(() => null);
      if (recovery?.required) {
        await onRecoveryRequired();
      }
      setResult({
        message: getErrorMessage(applyError),
        rolledBack: recovery?.required !== true,
        success: false,
      });
      setStep('result');
      setPhase('idle');
    }
  };

  const rebuildPreviewWithBackup = async () => {
    if (request === null) {
      return;
    }
    setBackupAndContinue(true);
    await loadPreview({ ...request, backupAndContinue: true });
  };

  const canContinueConfiguration = request !== null;
  const canContinueSafety = !isNewTarget || (targetConfirmed && warningAcknowledged);
  const canApply = preview?.canApply === true && phase === 'idle';

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (step === 'target') {
      setStep('configuration');
    } else if (step === 'configuration' && canContinueConfiguration) {
      setStep('safety');
    } else if (step === 'safety' && canContinueSafety) {
      await loadPreview();
    } else if (step === 'preview' && canApply) {
      await apply();
    }
  };

  const goBack = () => {
    if (phase !== 'idle') {
      return;
    }
    const currentIndex = steps.indexOf(step);
    if (currentIndex > 0 && step !== 'result') {
      setStep(steps[currentIndex - 1]);
    }
  };

  return (
    <section className="optiscaler-wizard" aria-label={`${actionLabel(selection.action)} OptiScaler`}>
      <div className="optiscaler-wizard-header">
        <div>
          <h2>{actionLabel(selection.action)} OptiScaler</h2>
          <p>{targetRelativePath}</p>
        </div>
        <button disabled={phase !== 'idle'} onClick={onClose} type="button">Close</button>
      </div>

      <ol className="optiscaler-wizard-steps" aria-label="OptiScaler management steps">
        {steps.map((wizardStep, index) => (
          <li className={wizardStep === step ? 'optiscaler-wizard-step-active' : ''} key={wizardStep}>
            <span>{index + 1}</span>
            {stepLabels[wizardStep]}
          </li>
        ))}
      </ol>

      <form className="optiscaler-wizard-form" onSubmit={submit}>
        {step === 'target' && (
          <div className="optiscaler-wizard-content">
            <h3>Selected target</h3>
            <dl className="optiscaler-wizard-summary">
              <div><dt>Directory</dt><dd>{targetRelativePath}</dd></div>
              <div><dt>Executable</dt><dd>{executableRelativePath}</dd></div>
              <div><dt>Architecture</dt><dd>{selection.candidate?.architecture ?? 'x64'}</dd></div>
              <div><dt>Action</dt><dd>{actionLabel(selection.action)}</dd></div>
            </dl>
            {selection.candidate !== null && (
              <p className="optiscaler-wizard-note">
                Candidate ranking is evidence for target selection, not a guarantee that OptiScaler is compatible with this game.
              </p>
            )}
          </div>
        )}

        {step === 'configuration' && (
          <div className="optiscaler-wizard-content">
            <h3>Configuration</h3>
            {isUninstall ? (
              <dl className="optiscaler-wizard-summary">
                <div><dt>Graphics API</dt><dd>{selection.target?.GraphicsAPI}</dd></div>
                <div><dt>Proxy</dt><dd>{selection.target?.ProxyFilename}</dd></div>
                <div><dt>Process filter</dt><dd>{selection.target?.ProcessFilter ?? 'Cleared'}</dd></div>
              </dl>
            ) : (
              <div className="optiscaler-wizard-fields">
                <label>
                  Graphics API
                  <select
                    onChange={(event) => chooseGraphicsAPI(event.target.value as GraphicsAPI | '')}
                    value={graphicsAPI}
                  >
                    <option value="">Choose an API</option>
                    <option value={GraphicsAPI.GraphicsAPIDirectX}>DirectX</option>
                    <option value={GraphicsAPI.GraphicsAPIVulkan}>Vulkan</option>
                  </select>
                </label>
                <label>
                  Proxy filename
                  <select
                    disabled={graphicsAPI === ''}
                    onChange={(event) => setProxyFilename(event.target.value)}
                    value={proxyFilename}
                  >
                    {supportedProxyFilenames.map((filename) => (
                      <option key={filename} value={filename}>{filename}</option>
                    ))}
                  </select>
                  {graphicsAPI !== '' && (
                    <span>
                      Recommended: {graphicsAPI === GraphicsAPI.GraphicsAPIDirectX ? 'dxgi.dll' : 'winmm.dll'}
                    </span>
                  )}
                </label>
                <label>
                  DXGI spoofing
                  <select
                    onChange={(event) => setDXGISpoofing(
                      event.target.value === '' ? null : event.target.value === 'true',
                    )}
                    value={dxgiSpoofing === null ? '' : String(dxgiSpoofing)}
                  >
                    <option value="">Choose a setting</option>
                    <option value="false">Disabled</option>
                    <option value="true">Enabled</option>
                  </select>
                </label>
                <label>
                  Process filter
                  <input
                    onChange={(event) => setProcessFilter(event.target.value)}
                    placeholder="Leave empty for a shared executable directory"
                    type="text"
                    value={processFilter}
                  />
                  <span>Defaults to the selected executable and may be cleared.</span>
                </label>
                {hasDetectedReShade && (
                  <label className="optiscaler-wizard-checkbox">
                    <input
                      checked={enableReShadeCoexistence}
                      disabled={graphicsAPI === GraphicsAPI.GraphicsAPIVulkan}
                      onChange={(event) => setEnableReShadeCoexistence(event.target.checked)}
                      type="checkbox"
                    />
                    Chain the detected ReShade runtime through OptiScaler
                  </label>
                )}
                {graphicsAPI === GraphicsAPI.GraphicsAPIVulkan && hasDetectedReShade && (
                  <p className="optiscaler-wizard-error">
                    Automated Vulkan and ReShade coexistence is not supported.
                  </p>
                )}
              </div>
            )}
          </div>
        )}

        {step === 'safety' && (
          <div className="optiscaler-wizard-content">
            <h3>Confirm before preview</h3>
            {isNewTarget ? (
              <>
                <label className="optiscaler-wizard-checkbox">
                  <input
                    checked={targetConfirmed}
                    onChange={(event) => setTargetConfirmed(event.target.checked)}
                    type="checkbox"
                  />
                  I confirm that {executableRelativePath} and {proxyFilename} are correct.
                </label>
                <div className="optiscaler-wizard-warning">
                  <p>
                    OptiScaler can be incompatible with online games and anti-cheat systems. Candidate ranking and
                    upstream compatibility reports do not guarantee that this game is safe or supported.
                  </p>
                  <a
                    href="https://github.com/optiscaler/OptiScaler/wiki/Compatibility-List"
                    rel="noreferrer"
                    target="_blank"
                  >
                    Review upstream compatibility guidance
                  </a>
                </div>
                <label className="optiscaler-wizard-checkbox">
                  <input
                    checked={warningAcknowledged}
                    onChange={(event) => setWarningAcknowledged(event.target.checked)}
                    type="checkbox"
                  />
                  I understand the online-game and anti-cheat risk.
                </label>
              </>
            ) : (
              <p className="optiscaler-wizard-note">
                Fiach will recalculate the backend preview immediately before apply and reject stale changes.
              </p>
            )}
          </div>
        )}

        {step === 'preview' && preview !== null && (
          <div className="optiscaler-wizard-content">
            <h3>Review planned changes</h3>
            <OptiScalerPreview preview={preview} />
            {preview.drift.length > 0 && !backupAndContinue && (
              <div className="optiscaler-wizard-drift-actions">
                <p>Drifted files can only be cancelled or archived before continuing.</p>
                <button
                  disabled={phase !== 'idle'}
                  onClick={() => void rebuildPreviewWithBackup()}
                  type="button"
                >
                  Back up drift and rebuild preview
                </button>
              </div>
            )}
          </div>
        )}

        {step === 'result' && result !== null && (
          <div className="optiscaler-wizard-content">
            <h3>{result.success ? 'Operation complete' : result.rolledBack ? 'Operation rolled back' : 'Recovery required'}</h3>
            <p className="optiscaler-wizard-note">{result.message}</p>
            {phase === 'refreshing' && <p className="optiscaler-wizard-note">Refreshing target state...</p>}
          </div>
        )}

        {error !== null && <p className="optiscaler-wizard-error">{error}</p>}

        <div className="optiscaler-wizard-footer">
          {step !== 'target' && step !== 'result' && (
            <button disabled={phase !== 'idle'} onClick={goBack} type="button">Back</button>
          )}
          {step !== 'result' && (
            <button
              className="button-main"
              disabled={
                phase !== 'idle' ||
                (step === 'configuration' && !canContinueConfiguration) ||
                (step === 'safety' && !canContinueSafety) ||
                (step === 'preview' && !canApply)
              }
              type="submit"
            >
              {phase === 'previewing'
                ? 'Building preview...'
                : phase === 'applying'
                  ? 'Applying...'
                  : step === 'safety'
                    ? 'Preview'
                    : step === 'preview'
                      ? actionLabel(selection.action)
                      : 'Next'}
            </button>
          )}
          {step === 'result' && (
            <button className="button-main" disabled={phase !== 'idle'} onClick={onClose} type="button">
              Done
            </button>
          )}
        </div>
      </form>
    </section>
  );
};
