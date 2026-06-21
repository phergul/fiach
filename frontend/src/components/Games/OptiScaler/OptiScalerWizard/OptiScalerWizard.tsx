import { FormEvent, useEffect, useMemo, useState } from 'react';

import { X } from 'lucide-react';

import { Action, GraphicsAPI } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import type {
  OptiScalerApplyResult,
  OptiScalerCandidate,
  OptiScalerPreview as OptiScalerPreviewModel,
  OptiScalerRequest,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ApplyOptiScalerAction,
  GetOptiScalerRecoveryState,
  PreviewOptiScalerAction,
} from '@bindings/github.com/phergul/fiach/internal/services/optiscalerservice';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { OptiScalerPreview } from '@components/Games/OptiScaler/OptiScalerPreview/OptiScalerPreview';
import { OptiScalerWizardConfigurationStep } from './OptiScalerWizardConfigurationStep/OptiScalerWizardConfigurationStep';
import { OptiScalerWizardSafetyStep } from './OptiScalerWizardSafetyStep/OptiScalerWizardSafetyStep';
import { OptiScalerWizardTargetStep } from './OptiScalerWizardTargetStep/OptiScalerWizardTargetStep';
import { getErrorMessage } from '@utils';

import './OptiScalerWizard.scss';

type WizardStep = 'target' | 'configuration' | 'safety' | 'preview' | 'result';
type OperationPhase = 'idle' | 'previewing' | 'applying' | 'refreshing';

interface OperationDefinition {
  label: string;
  steps: WizardStep[];
}

interface OptiScalerWizardProps {
  gameID: number;
  onClose: () => void;
  onRecoveryRequired: () => Promise<void>;
  onRefresh: () => Promise<void>;
  selection: OptiScalerOperationSelection;
}

export interface OptiScalerOperationSelection {
  action: Action;
  candidate: OptiScalerCandidate | null;
  target: OptiScalerTarget | null;
}

const stepLabels: Record<WizardStep, string> = {
  target: 'Target',
  configuration: 'Configuration',
  safety: 'Safety',
  preview: 'Preview',
  result: 'Result',
};

const operationDefinitions: Record<Action, OperationDefinition> = {
  [Action.$zero]: { label: 'Manage', steps: ['target', 'preview', 'result'] },
  [Action.ActionInstall]: {
    label: 'Install',
    steps: ['target', 'configuration', 'safety', 'preview', 'result'],
  },
  [Action.ActionAdopt]: {
    label: 'Adopt',
    steps: ['target', 'configuration', 'safety', 'preview', 'result'],
  },
  [Action.ActionUpdate]: {
    label: 'Update',
    steps: ['configuration', 'preview', 'result'],
  },
  [Action.ActionRepair]: {
    label: 'Repair',
    steps: ['configuration', 'preview', 'result'],
  },
  [Action.ActionUninstall]: {
    label: 'Uninstall',
    steps: ['target', 'preview', 'result'],
  },
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

interface WizardValues {
  dxgiSpoofing: boolean | null;
  enableReShadeCoexistence: boolean;
  graphicsAPI: GraphicsAPI | '';
  processFilter: string;
  proxyFilename: string;
  targetConfirmed: boolean;
  warningAcknowledged: boolean;
}

const initialValues = (selection: OptiScalerOperationSelection): WizardValues => {
  const executableRelativePath =
    selection.candidate?.executableRelativePath ?? selection.target?.ExecutableRelativePath ?? '';
  const executableName = selection.candidate?.executableName
    ?? executableRelativePath.split(/[\\/]/).pop()
    ?? '';
  const storedGraphicsAPI = selection.target?.GraphicsAPI;

  return {
    dxgiSpoofing: selection.target === null ? null : selection.target.DXGISpoofing,
    enableReShadeCoexistence: selection.target?.EnableReShadeCoexistence ?? false,
    graphicsAPI: storedGraphicsAPI === GraphicsAPI.GraphicsAPIVulkan
      ? GraphicsAPI.GraphicsAPIVulkan
      : storedGraphicsAPI === GraphicsAPI.GraphicsAPIDirectX
        ? GraphicsAPI.GraphicsAPIDirectX
        : '',
    processFilter: selection.target === null ? executableName : selection.target.ProcessFilter ?? '',
    proxyFilename: selection.target?.ProxyFilename ?? '',
    targetConfirmed: false,
    warningAcknowledged: false,
  };
};

export const OptiScalerWizard = ({
  gameID,
  onClose,
  onRecoveryRequired,
  onRefresh,
  selection,
}: OptiScalerWizardProps) => {
  const definition = operationDefinitions[selection.action];
  const initial = useMemo(() => initialValues(selection), [selection]);
  const executableRelativePath =
    selection.candidate?.executableRelativePath ?? selection.target?.ExecutableRelativePath ?? '';
  const targetRelativePath =
    selection.candidate?.targetRelativePath ?? selection.target?.TargetRelativePath ?? '';
  const executableName =
    selection.candidate?.executableName ?? executableRelativePath.split(/[\\/]/).pop() ?? '';
  const [step, setStep] = useState<WizardStep>(definition.steps[0]);
  const [values, setValues] = useState<WizardValues>(initial);
  const [backupAndContinue, setBackupAndContinue] = useState(false);
  const [preview, setPreview] = useState<OptiScalerPreviewModel | null>(null);
  const [result, setResult] = useState<OptiScalerApplyResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [phase, setPhase] = useState<OperationPhase>('idle');
  const [isDiscardOpen, setIsDiscardOpen] = useState(false);
  const currentStepIndex = definition.steps.indexOf(step);
  const isNewTarget = selection.action === Action.ActionInstall || selection.action === Action.ActionAdopt;
  const isDirty = JSON.stringify(values) !== JSON.stringify(initial) || backupAndContinue;

  useEffect(() => {
    setStep(definition.steps[0]);
    setValues(initial);
    setBackupAndContinue(false);
    setPreview(null);
    setResult(null);
    setError(null);
    setPhase('idle');
    setIsDiscardOpen(false);
  }, [definition.steps, initial, selection]);

  const updateValues = (nextValues: Partial<WizardValues>) => {
    setValues((current) => ({ ...current, ...nextValues }));
    setPreview(null);
  };

  const request = useMemo<OptiScalerRequest | null>(() => {
    if (values.graphicsAPI === '' || values.dxgiSpoofing === null || values.proxyFilename === '') {
      return null;
    }
    return {
      acknowledgeWarning: isNewTarget && values.warningAcknowledged,
      action: selection.action,
      backupAndContinue,
      dxgiSpoofing: values.dxgiSpoofing,
      enableReShadeCoexistence: values.enableReShadeCoexistence,
      executableRelativePath,
      gameId: gameID,
      graphicsApi: values.graphicsAPI,
      processFilter: values.processFilter.trim() === '' ? null : values.processFilter.trim(),
      proxyFilename: values.proxyFilename,
      targetRelativePath,
    };
  }, [
    backupAndContinue,
    executableRelativePath,
    gameID,
    isNewTarget,
    selection.action,
    targetRelativePath,
    values,
  ]);

  const chooseGraphicsAPI = (nextAPI: GraphicsAPI | '') => {
    updateValues({
      enableReShadeCoexistence: nextAPI === GraphicsAPI.GraphicsAPIVulkan
        ? false
        : values.enableReShadeCoexistence,
      graphicsAPI: nextAPI,
      proxyFilename: nextAPI === GraphicsAPI.GraphicsAPIDirectX
        ? 'dxgi.dll'
        : nextAPI === GraphicsAPI.GraphicsAPIVulkan
          ? 'winmm.dll'
          : '',
    });
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

  const canContinue = step === 'configuration'
    ? request !== null
    : step === 'safety'
      ? values.targetConfirmed && values.warningAcknowledged
      : step === 'preview'
        ? preview?.canApply === true
        : true;

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!canContinue || phase !== 'idle') {
      return;
    }
    if (step === 'preview') {
      await apply();
      return;
    }
    const nextStep = definition.steps[currentStepIndex + 1];
    if (nextStep === 'preview') {
      await loadPreview();
    } else if (nextStep !== undefined) {
      setStep(nextStep);
    }
  };

  const goBack = () => {
    if (phase === 'idle' && currentStepIndex > 0 && step !== 'result') {
      setStep(definition.steps[currentStepIndex - 1]);
    }
  };

  const requestClose = () => {
    if (phase !== 'idle') {
      return;
    }
    if (isDirty && step !== 'result') {
      setIsDiscardOpen(true);
      return;
    }
    onClose();
  };

  const primaryLabel = phase === 'previewing'
    ? 'Building preview...'
    : phase === 'applying'
      ? 'Applying...'
      : step === 'preview'
        ? definition.label
        : definition.steps[currentStepIndex + 1] === 'preview'
          ? 'Preview'
          : 'Next';

  return (
    <>
      <section className="optiscaler-wizard" aria-label={`${definition.label} OptiScaler`}>
        <header className="optiscaler-wizard-header">
          <div>
            <h2>{definition.label} OptiScaler</h2>
            <p>{executableName}</p>
          </div>
          <button
            aria-label="Close OptiScaler wizard"
            disabled={phase !== 'idle'}
            onClick={requestClose}
            type="button"
          >
            <X aria-hidden="true" />
          </button>
        </header>

        <ol
          className="optiscaler-wizard-steps"
          aria-label="OptiScaler management steps"
          style={{ gridTemplateColumns: `repeat(${definition.steps.length}, minmax(0, 1fr))` }}
        >
          {definition.steps.map((wizardStep, index) => (
            <li
              className={index < currentStepIndex
                ? 'optiscaler-wizard-step-complete'
                : wizardStep === step
                  ? 'optiscaler-wizard-step-active'
                  : ''}
              key={wizardStep}
            >
              {index + 1}. {stepLabels[wizardStep]}
            </li>
          ))}
        </ol>

        <form className="optiscaler-wizard-form" onSubmit={submit}>
          <div className="optiscaler-wizard-scroll">
            {step === 'target' && (
              <OptiScalerWizardTargetStep actionLabel={definition.label} selection={selection} />
            )}
            {step === 'configuration' && (
              <OptiScalerWizardConfigurationStep
                dxgiSpoofing={values.dxgiSpoofing}
                enableReShadeCoexistence={values.enableReShadeCoexistence}
                graphicsAPI={values.graphicsAPI}
                hasDetectedReShade={selection.candidate?.hasReShade ?? false}
                onChooseGraphicsAPI={chooseGraphicsAPI}
                onDXGISpoofingChange={(value) => updateValues({ dxgiSpoofing: value })}
                onProcessFilterChange={(value) => updateValues({ processFilter: value })}
                onProxyFilenameChange={(value) => updateValues({ proxyFilename: value })}
                onReShadeCoexistenceChange={(value) => updateValues({ enableReShadeCoexistence: value })}
                processFilter={values.processFilter}
                proxyFilename={values.proxyFilename}
                supportedProxyFilenames={supportedProxyFilenames}
              />
            )}
            {step === 'safety' && (
              <OptiScalerWizardSafetyStep
                executableRelativePath={executableRelativePath}
                onTargetConfirmedChange={(value) => updateValues({ targetConfirmed: value })}
                onWarningAcknowledgedChange={(value) => updateValues({ warningAcknowledged: value })}
                proxyFilename={values.proxyFilename}
                targetConfirmed={values.targetConfirmed}
                warningAcknowledged={values.warningAcknowledged}
              />
            )}
            {step === 'preview' && preview !== null && (
              <div className="optiscaler-wizard-content">
                <dl className="optiscaler-wizard-summary optiscaler-wizard-preview-summary">
                  <div><dt>Executable</dt><dd>{executableName}</dd></div>
                  <div>
                    <dt>Version</dt>
                    <dd>
                      {preview.release?.version
                        || preview.release?.tag
                        || selection.target?.ReleaseVersion
                        || selection.target?.ReleaseTag
                        || 'Latest stable'}
                    </dd>
                  </div>
                </dl>
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
                <div className={result.success
                  ? 'optiscaler-wizard-result optiscaler-wizard-result-success'
                  : 'optiscaler-wizard-result optiscaler-wizard-result-error'}
                >
                  <h3>
                    {result.success
                      ? 'Operation complete'
                      : result.rolledBack
                        ? 'Operation rolled back'
                        : 'Recovery required'}
                  </h3>
                  <p>{result.message}</p>
                  {phase === 'refreshing' && <p>Refreshing target state...</p>}
                </div>
              </div>
            )}
            {error !== null && <p className="optiscaler-wizard-error">{error}</p>}
          </div>

          <footer className="optiscaler-wizard-footer">
            {currentStepIndex > 0 && step !== 'result' && (
              <button disabled={phase !== 'idle'} onClick={goBack} type="button">Back</button>
            )}
            {step !== 'result' ? (
              <button className="button-main" disabled={!canContinue || phase !== 'idle'} type="submit">
                {primaryLabel}
              </button>
            ) : (
              <button className="button-main" disabled={phase !== 'idle'} onClick={onClose} type="button">
                Done
              </button>
            )}
          </footer>
        </form>
      </section>

      <ConfirmDialog
        confirmLabel="Discard changes"
        isOpen={isDiscardOpen}
        message="Your changes to this OptiScaler operation will be discarded."
        onCancel={() => setIsDiscardOpen(false)}
        onConfirm={onClose}
        title="Discard OptiScaler changes?"
      />
    </>
  );
};
