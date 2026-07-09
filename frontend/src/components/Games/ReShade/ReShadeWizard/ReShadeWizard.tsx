import { FormEvent, useEffect, useMemo, useState } from 'react';

import { X } from 'lucide-react';

import {
  Action,
  Architecture,
  BuildVariant,
  type ContentRequest,
  RenderingAPI,
  type Preview as ReShadePreviewModel,
  type ApplyResult as ReShadeApplyResult,
} from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type {
  ReShadeChainTarget,
  ReShadeContentCatalogue,
  ReShadePresetInspectionResult,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ApplyReShadeAction,
  GetReShadeRecoveryState,
  InspectReShadePreset,
  ListReShadeContentCatalogue,
  PreviewReShadeAction,
} from '@bindings/github.com/phergul/fiach/internal/services/reshadeservice';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { WizardError } from '@components/Common/WizardError/WizardError';
import { ReShadePreview } from '@components/Games/ReShade/ReShadePreview/ReShadePreview';
import type { ReShadeOperationSelection } from '@components/Games/ReShade/ReShadeTargetTable/ReShadeTargetTable';
import { ReShadeWizardContentStep } from './ReShadeWizardContentStep/ReShadeWizardContentStep';
import { ReShadeWizardRuntimeStep } from './ReShadeWizardRuntimeStep/ReShadeWizardRuntimeStep';
import { ReShadeWizardSafetyStep } from './ReShadeWizardSafetyStep/ReShadeWizardSafetyStep';
import { ReShadeWizardTargetStep } from './ReShadeWizardTargetStep/ReShadeWizardTargetStep';
import { getErrorMessage } from '@utils';

import './ReShadeWizard.scss';

type WizardStep = 'target' | 'runtime' | 'content' | 'safety' | 'preview' | 'result';
type OperationPhase = 'idle' | 'previewing' | 'applying' | 'refreshing' | 'inspecting';

interface WizardErrorState {
  details: string;
  summary: string;
}

interface ReShadeWizardProps {
  catalogue: ReShadeContentCatalogue | null;
  chainTargets: ReShadeChainTarget[];
  gameID: number;
  onClose: () => void;
  onRecoveryRequired: () => Promise<void>;
  onRefresh: () => Promise<void>;
  selection: ReShadeOperationSelection;
}

interface WizardValues {
  antiCheatRiskAcknowledged: boolean;
  buildVariant: BuildVariant;
  content: ContentRequest;
  proxyFilename: string;
  renderingAPI: RenderingAPI | '';
  singlePlayerAcknowledged: boolean;
}

const stepLabels: Record<WizardStep, string> = {
  target: 'Target',
  runtime: 'Runtime',
  content: 'Content',
  safety: 'Safety',
  preview: 'Preview',
  result: 'Result',
};

const operationLabel: Record<Action, string> = {
  [Action.$zero]: 'Manage',
  [Action.ActionInstall]: 'Install',
  [Action.ActionAdopt]: 'Adopt',
  [Action.ActionUpdate]: 'Update',
  [Action.ActionRepair]: 'Repair',
  [Action.ActionUninstall]: 'Uninstall',
  [Action.ActionConfigureContent]: 'Configure content',
};

const filename = (path: string) => path.split(/[\\/]/).pop() ?? path;

const initialValues = (selection: ReShadeOperationSelection): WizardValues => {
  const firstAPI = selection.candidate?.apiOptions[0] ?? null;
  const renderingAPI = selection.target?.RenderingAPI ?? firstAPI?.renderingApi ?? '';
  return {
    antiCheatRiskAcknowledged: selection.target?.BuildVariant === BuildVariant.BuildVariantAddon,
    buildVariant: selection.target?.BuildVariant ?? BuildVariant.BuildVariantStandard,
    content: {},
    proxyFilename: selection.target?.ProxyFilename ?? firstAPI?.proxies[0] ?? '',
    renderingAPI,
    singlePlayerAcknowledged: selection.target?.BuildVariant === BuildVariant.BuildVariantAddon,
  };
};

const stepsFor = (action: Action, buildVariant: BuildVariant): WizardStep[] => {
  if (action === Action.ActionUninstall) {
    return ['target', 'preview', 'result'];
  }
  if (action === Action.ActionConfigureContent) {
    return ['content', 'preview', 'result'];
  }
  const steps: WizardStep[] =
    action === Action.ActionInstall || action === Action.ActionAdopt
      ? ['target', 'runtime', 'content']
      : ['runtime', 'content'];
  if (buildVariant === BuildVariant.BuildVariantAddon) {
    steps.push('safety');
  }
  return [...steps, 'preview', 'result'];
};

const hasContent = (content: ContentRequest) =>
  (content.effectPackages?.length ?? 0) > 0 || (content.addons?.length ?? 0) > 0;

const selectionIdentity = (selection: ReShadeOperationSelection) => {
  if (selection.target !== null) {
    return `target:${selection.target.ID}:${selection.action}:${selection.target.BuildVariant}:${selection.target.RenderingAPI}:${selection.target.ProxyFilename}`;
  }
  if (selection.candidate !== null) {
    const firstAPI = selection.candidate.apiOptions[0];
    const apiIdentity =
      firstAPI !== undefined
        ? `${firstAPI.renderingApi}:${firstAPI.proxies[0] ?? ''}`
        : '';
    return `candidate:${selection.candidate.targetRelativePath}:${selection.candidate.executableRelativePath}:${selection.action}:${apiIdentity}`;
  }
  return `action:${selection.action}`;
};

export const ReShadeWizard = ({
  catalogue,
  chainTargets,
  gameID,
  onClose,
  onRecoveryRequired,
  onRefresh,
  selection,
}: ReShadeWizardProps) => {
  const initial = useMemo(() => initialValues(selection), [selection]);
  const activeSelectionIdentity = useMemo(() => selectionIdentity(selection), [selection]);
  const executableRelativePath =
    selection.candidate?.executableRelativePath ?? selection.target?.ExecutableRelativePath ?? '';
  const targetRelativePath =
    selection.candidate?.targetRelativePath ?? selection.target?.TargetRelativePath ?? '';
  const architecture =
    selection.candidate?.architecture ??
    selection.target?.Architecture ??
    Architecture.ArchitectureX64;
  const [values, setValues] = useState<WizardValues>(initial);
  const steps = useMemo(
    () => stepsFor(selection.action, values.buildVariant),
    [selection.action, values.buildVariant],
  );
  const [step, setStep] = useState<WizardStep>(steps[0]);
  const [backupAndContinue, setBackupAndContinue] = useState(false);
  const [preview, setPreview] = useState<ReShadePreviewModel | null>(null);
  const [result, setResult] = useState<ReShadeApplyResult | null>(null);
  const [error, setError] = useState<WizardErrorState | null>(null);
  const [phase, setPhase] = useState<OperationPhase>('idle');
  const [isDiscardOpen, setIsDiscardOpen] = useState(false);
  const [presetPath, setPresetPath] = useState('');
  const [inspection, setInspection] = useState<ReShadePresetInspectionResult | null>(null);
  const [currentCatalogue, setCurrentCatalogue] = useState<ReShadeContentCatalogue | null>(
    catalogue,
  );
  const currentStepIndex = steps.indexOf(step);
  const chainTarget =
    chainTargets.find((target) => target.TargetRelativePath === targetRelativePath) ?? null;
  const apiOptions = selection.candidate?.apiOptions ?? [
    {
      renderingApi: selection.target?.RenderingAPI ?? RenderingAPI.RenderingAPID3D11,
      proxies: [selection.target?.ProxyFilename ?? 'dxgi.dll'],
    },
  ];

  useEffect(() => {
    const nextInitial = initialValues(selection);
    setValues(nextInitial);
    setStep(stepsFor(selection.action, nextInitial.buildVariant)[0]);
    setBackupAndContinue(false);
    setPreview(null);
    setResult(null);
    setError(null);
    setPhase('idle');
    setIsDiscardOpen(false);
    setPresetPath('');
    setInspection(null);
  }, [activeSelectionIdentity]);

  useEffect(() => {
    setCurrentCatalogue(catalogue);
  }, [catalogue]);

  const updateValues = (nextValues: Partial<WizardValues>) => {
    setValues((current) => ({ ...current, ...nextValues }));
    setPreview(null);
  };

  const request = useMemo(() => {
    if (values.renderingAPI === '' || values.proxyFilename === '') {
      return null;
    }
    return {
      action: selection.action,
      antiCheatRiskAcknowledged: values.antiCheatRiskAcknowledged,
      architecture,
      backupAndContinue,
      buildVariant: values.buildVariant,
      content: hasContent(values.content) ? values.content : undefined,
      executableRelativePath,
      gameId: gameID,
      proxyFilename: values.proxyFilename,
      renderingApi: values.renderingAPI,
      singlePlayerAcknowledged: values.singlePlayerAcknowledged,
      targetRelativePath,
    };
  }, [
    architecture,
    backupAndContinue,
    executableRelativePath,
    gameID,
    selection.action,
    targetRelativePath,
    values,
  ]);

  const chooseRenderingAPI = (nextAPI: RenderingAPI) => {
    const option = apiOptions.find((item) => item.renderingApi === nextAPI);
    updateValues({
      proxyFilename: option?.proxies[0] ?? '',
      renderingAPI: nextAPI,
    });
  };

  const loadPreview = async () => {
    if (request === null) {
      return;
    }
    setPhase('previewing');
    setError(null);
    try {
      setPreview(await PreviewReShadeAction(request));
      setStep('preview');
    } catch (previewError) {
      setError({
        details: getErrorMessage(previewError),
        summary: 'Could not build the ReShade preview.',
      });
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
      const applyResult = await ApplyReShadeAction(request, preview.previewHash);
      setResult(applyResult);
      setStep('result');
      setPhase('refreshing');
      await onRefresh();
      setPhase('idle');
    } catch (applyError) {
      const message = getErrorMessage(applyError);
      setError({
        details: message,
        summary: 'Could not apply the ReShade operation.',
      });
      const recovery = await GetReShadeRecoveryState().catch(() => null);
      if (recovery?.required) {
        await onRecoveryRequired();
      }
      setResult({
        message,
        rolledBack: recovery?.required !== true,
        success: false,
      });
      setStep('result');
      setPhase('idle');
    }
  };

  const refreshCatalogue = async () => {
    setPhase('refreshing');
    try {
      setCurrentCatalogue(await ListReShadeContentCatalogue(true));
    } catch (catalogueError) {
      setError({
        details: getErrorMessage(catalogueError),
        summary: 'Could not refresh the ReShade catalogue.',
      });
    } finally {
      setPhase('idle');
    }
  };

  const inspectPreset = async (path: string) => {
    setPhase('inspecting');
    setError(null);
    try {
      setInspection(await InspectReShadePreset(gameID, targetRelativePath, path));
    } catch (inspectError) {
      setError({
        details: getErrorMessage(inspectError),
        summary: 'Could not inspect the ReShade preset.',
      });
    } finally {
      setPhase('idle');
    }
  };

  const rebuildPreviewWithBackup = async () => {
    setBackupAndContinue(true);
    if (request !== null) {
      setPhase('previewing');
      setError(null);
      try {
        setPreview(await PreviewReShadeAction({ ...request, backupAndContinue: true }));
      } catch (previewError) {
        setError({
          details: getErrorMessage(previewError),
          summary: 'Could not rebuild the ReShade preview.',
        });
      } finally {
        setPhase('idle');
      }
    }
  };

  const canContinue =
    step === 'runtime'
      ? request !== null
      : step === 'safety'
        ? values.singlePlayerAcknowledged && values.antiCheatRiskAcknowledged
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
    const nextStep = steps[currentStepIndex + 1];
    if (nextStep === 'preview') {
      await loadPreview();
    } else if (nextStep !== undefined) {
      setStep(nextStep);
    }
  };

  const goBack = () => {
    if (phase === 'idle' && currentStepIndex > 0 && step !== 'result') {
      setStep(steps[currentStepIndex - 1]);
    }
  };

  const requestClose = () => {
    if (phase !== 'idle') {
      return;
    }
    if (
      step !== 'result' &&
      (preview !== null || hasContent(values.content) || backupAndContinue)
    ) {
      setIsDiscardOpen(true);
      return;
    }
    onClose();
  };

  const primaryLabel =
    phase === 'previewing'
      ? 'Building preview...'
      : phase === 'applying'
        ? 'Applying...'
        : step === 'preview'
          ? operationLabel[selection.action]
          : steps[currentStepIndex + 1] === 'preview'
            ? 'Preview'
            : 'Next';

  return (
    <>
      <section
        className="reshade-wizard"
        aria-label={`${operationLabel[selection.action]} ReShade`}
      >
        <header className="reshade-wizard-header">
          <div>
            <h2>{operationLabel[selection.action]} ReShade</h2>
            <p>{filename(executableRelativePath)}</p>
          </div>
          <button
            aria-label="Close ReShade wizard"
            disabled={phase !== 'idle'}
            onClick={requestClose}
            type="button"
          >
            <X aria-hidden="true" />
          </button>
        </header>

        <ol
          className="reshade-wizard-steps"
          aria-label="ReShade management steps"
          style={{ gridTemplateColumns: `repeat(${steps.length}, minmax(0, 1fr))` }}
        >
          {steps.map((wizardStep, index) => (
            <li
              className={
                index < currentStepIndex
                  ? 'reshade-wizard-step-complete'
                  : wizardStep === step
                    ? 'reshade-wizard-step-active'
                    : ''
              }
              key={wizardStep}
            >
              {index + 1}. {stepLabels[wizardStep]}
            </li>
          ))}
        </ol>

        <form className="reshade-wizard-form" onSubmit={submit}>
          {error !== null && (
            <WizardError
              details={error.details}
              onClose={() => setError(null)}
              summary={error.summary}
            />
          )}
          <div
            className={
              step === 'content'
                ? 'reshade-wizard-scroll reshade-wizard-scroll-content-step'
                : 'reshade-wizard-scroll'
            }
          >
            {step === 'target' && <ReShadeWizardTargetStep selection={selection} />}
            {step === 'runtime' && (
              <ReShadeWizardRuntimeStep
                apiOptions={apiOptions}
                buildVariant={values.buildVariant}
                onBuildVariantChange={(value) => updateValues({ buildVariant: value })}
                onProxyFilenameChange={(value) => updateValues({ proxyFilename: value })}
                onRenderingAPIChange={chooseRenderingAPI}
                proxyFilename={values.proxyFilename}
                renderingAPI={values.renderingAPI}
              />
            )}
            {step === 'content' && (
              <ReShadeWizardContentStep
                buildVariant={values.buildVariant}
                catalogue={currentCatalogue}
                content={values.content}
                inspection={inspection}
                isInspectingPreset={phase === 'inspecting'}
                onContentChange={(content) => updateValues({ content })}
                onInspectPreset={(path) => void inspectPreset(path)}
                onRefreshCatalogue={() => void refreshCatalogue()}
                presetPath={presetPath}
                setPresetPath={setPresetPath}
              />
            )}
            {step === 'safety' && (
              <ReShadeWizardSafetyStep
                antiCheatRiskAcknowledged={values.antiCheatRiskAcknowledged}
                onAntiCheatRiskAcknowledgedChange={(value) =>
                  updateValues({ antiCheatRiskAcknowledged: value })
                }
                onSinglePlayerAcknowledgedChange={(value) =>
                  updateValues({ singlePlayerAcknowledged: value })
                }
                singlePlayerAcknowledged={values.singlePlayerAcknowledged}
              />
            )}
            {step === 'preview' && preview !== null && (
              <div className="reshade-wizard-content">
                <dl className="reshade-wizard-summary">
                  <div>
                    <dt>Executable</dt>
                    <dd>{filename(executableRelativePath)}</dd>
                  </div>
                  <div>
                    <dt>Proxy</dt>
                    <dd>{values.proxyFilename}</dd>
                  </div>
                  <div>
                    <dt>Build</dt>
                    <dd>{values.buildVariant === 'addon' ? 'Full add-on' : 'Standard'}</dd>
                  </div>
                </dl>
                <ReShadePreview chainTarget={chainTarget} preview={preview} />
                {preview.drift.length > 0 && !backupAndContinue && (
                  <div className="reshade-wizard-drift-actions">
                    <p>Drifted files can be cancelled or archived before continuing.</p>
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
              <div className="reshade-wizard-content">
                <div
                  className={
                    result.success
                      ? 'reshade-wizard-result reshade-wizard-result-success'
                      : 'reshade-wizard-result reshade-wizard-result-error'
                  }
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
          </div>

          <footer className="reshade-wizard-footer">
            {currentStepIndex > 0 && step !== 'result' && (
              <button disabled={phase !== 'idle'} onClick={goBack} type="button">
                Back
              </button>
            )}
            {step !== 'result' ? (
              <button
                className="button-main"
                disabled={!canContinue || phase !== 'idle'}
                type="submit"
              >
                {primaryLabel}
              </button>
            ) : (
              <button
                className="button-main"
                disabled={phase !== 'idle'}
                onClick={onClose}
                type="button"
              >
                Done
              </button>
            )}
          </footer>
        </form>
      </section>

      <ConfirmDialog
        confirmLabel="Discard changes"
        isOpen={isDiscardOpen}
        message="Your changes to this ReShade operation will be discarded."
        onCancel={() => setIsDiscardOpen(false)}
        onConfirm={onClose}
        title="Discard ReShade changes?"
      />
    </>
  );
};
