import { FormEvent, useEffect, useState } from 'react';

import {
  StrategyType,
  type ModSourceType,
  type Preview,
  type StrategyDescriptor,
  type Tag,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  DetectImportTargets,
  ListImportStrategies,
  PreviewImportConfiguration,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import { Modal } from '@components/Common/Modal/Modal';
import { GameModImportWizardDetailsStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardDetailsStep/GameModImportWizardDetailsStep';
import { GameModImportWizardPreviewStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardPreviewStep/GameModImportWizardPreviewStep';
import { GameModImportWizardStrategyStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardStrategyStep/GameModImportWizardStrategyStep';
import { GameModImportWizardTargetStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardTargetStep/GameModImportWizardTargetStep';
import type { ModTagSelection } from '@components/Games/Details/Mods/ModTags/ModTagEditor/ModTagEditor';
import { getErrorMessage } from '@utils';

import './GameModImportWizard.scss';

type GameModImportWizardStep = 'details' | 'strategy' | 'target' | 'preview';

interface GameModImportWizardSubmitInput {
  name: string;
  strategyType: StrategyType;
  targetRelativePath: string;
  tags: ModTagSelection[];
}

interface GameModImportWizardProps {
  error: string | null;
  availableTags: Tag[];
  gameID: number;
  initialName: string;
  isBusy: boolean;
  isOpen: boolean;
  onClose: () => void;
  onImport: (input: GameModImportWizardSubmitInput) => Promise<void> | void;
  onImportAnotherAfterCompleteChange?: (enabled: boolean) => void;
  onReusePreviousSettingsChange?: (enabled: boolean) => void;
  importAnotherAfterComplete?: boolean;
  queuePosition?: { current: number; total: number } | null;
  reuseFromPrevious?: { strategyType: StrategyType; targetRelativePath: string } | null;
  reusePreviousSettings?: boolean;
  sourceLabel: string;
  sourcePath: string;
  sourceType: ModSourceType;
  suggestedStrategyType: StrategyType | null;
  targetPath: string;
}

const stepLabels: Record<GameModImportWizardStep, string> = {
  details: 'Details',
  strategy: 'Strategy',
  target: 'Target',
  preview: 'Preview',
};

const stepOrder: GameModImportWizardStep[] = ['details', 'strategy', 'target', 'preview'];

export const GameModImportWizard = ({
  error,
  availableTags,
  gameID,
  initialName,
  isBusy,
  isOpen,
  onClose,
  onImport,
  onImportAnotherAfterCompleteChange,
  onReusePreviousSettingsChange,
  importAnotherAfterComplete = false,
  queuePosition = null,
  reuseFromPrevious = null,
  reusePreviousSettings = false,
  sourceLabel,
  sourcePath,
  sourceType,
  suggestedStrategyType,
  targetPath,
}: GameModImportWizardProps) => {
  const [step, setStep] = useState<GameModImportWizardStep>('details');
  const [name, setName] = useState(initialName);
  const [selectedStrategyType, setSelectedStrategyType] = useState<StrategyType | null>(null);
  const [targetRelativePath, setTargetRelativePath] = useState('.');
  const [tags, setTags] = useState<ModTagSelection[]>([]);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [strategies, setStrategies] = useState<StrategyDescriptor[]>([]);
  const [strategyLoadError, setStrategyLoadError] = useState<string | null>(null);
  const [isLoadingStrategies, setIsLoadingStrategies] = useState(false);
  const [targetCandidates, setTargetCandidates] = useState<string[]>([]);
  const [targetDetectionWarnings, setTargetDetectionWarnings] = useState<string[]>([]);
  const [targetDetectionError, setTargetDetectionError] = useState<string | null>(null);
  const [isDetectingTargets, setIsDetectingTargets] = useState(false);
  const trimmedName = name.trim();
  const selectedStrategy = strategies.find((strategy) => strategy.Type === selectedStrategyType);
  const isDetailsStep = step === 'details';
  const isStrategyStep = step === 'strategy';
  const isTargetStep = step === 'target';
  const isPreviewStep = step === 'preview';
  const currentStepIndex = stepOrder.indexOf(step);
  const canContinueFromDetails = trimmedName !== '';
  const canContinueFromStrategy = selectedStrategyType !== null && strategyLoadError === null;
  const canPreviewTarget = selectedStrategyType !== null && targetRelativePath.trim() !== '';
  const isNextDisabled =
    isBusy ||
    isPreviewing ||
    isDetectingTargets ||
    (isDetailsStep && !canContinueFromDetails) ||
    (isStrategyStep && !canContinueFromStrategy) ||
    (isTargetStep && !canPreviewTarget) ||
    (isPreviewStep && preview === null);

  const canReusePreviousSettings =
    reuseFromPrevious !== null && onReusePreviousSettingsChange !== undefined;
  const queueIsland =
    queuePosition === null ? undefined : (
      <div className="game-mod-import-wizard-queue-island" aria-live="polite">
        Queue: {queuePosition.current} of {queuePosition.total}
      </div>
    );

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    let isCancelled = false;
    setStep('details');
    setName(initialName);
    setSelectedStrategyType(null);
    setTargetRelativePath(
      reusePreviousSettings && reuseFromPrevious !== null
        ? reuseFromPrevious.targetRelativePath
        : '.',
    );
    setTags([]);
    setPreview(null);
    setPreviewError(null);
    setIsPreviewing(false);
    setStrategyLoadError(null);
    setStrategies([]);
    setIsLoadingStrategies(true);
    setTargetCandidates([]);
    setTargetDetectionWarnings([]);
    setTargetDetectionError(null);
    setIsDetectingTargets(false);

    ListImportStrategies()
      .then((loadedStrategies) => {
        if (isCancelled) {
          return;
        }

        setStrategies(loadedStrategies);
        const preferredStrategyType =
          reusePreviousSettings && reuseFromPrevious !== null
            ? reuseFromPrevious.strategyType
            : suggestedStrategyType;
        if (
          preferredStrategyType !== null &&
          loadedStrategies.some((strategy) => strategy.Type === preferredStrategyType)
        ) {
          setSelectedStrategyType(preferredStrategyType);
        } else if (loadedStrategies.length === 1) {
          setSelectedStrategyType(loadedStrategies[0].Type);
        }
      })
      .catch((loadError) => {
        if (isCancelled) {
          return;
        }

        setStrategyLoadError(getErrorMessage(loadError));
      })
      .finally(() => {
        if (!isCancelled) {
          setIsLoadingStrategies(false);
        }
      });

    return () => {
      isCancelled = true;
    };
  }, [initialName, isOpen, reuseFromPrevious, reusePreviousSettings, suggestedStrategyType]);

  useEffect(() => {
    if (!isOpen || !isTargetStep || selectedStrategy === undefined) {
      return;
    }
    if (!selectedStrategy.SupportsTargetDetection) {
      setTargetCandidates([]);
      setTargetDetectionWarnings([]);
      setTargetDetectionError(null);
      setIsDetectingTargets(false);
      return;
    }

    let isCancelled = false;
    setTargetCandidates([]);
    setTargetDetectionWarnings([]);
    setTargetDetectionError(null);
    setIsDetectingTargets(true);

    DetectImportTargets(gameID, selectedStrategy.Type)
      .then((detection) => {
        if (isCancelled) {
          return;
        }

        setTargetCandidates(detection.Candidates);
        setTargetDetectionWarnings(detection.Warnings);
        if (detection.Candidates.length === 1) {
          setTargetRelativePath((currentPath) =>
            currentPath.trim() === '' ? detection.Candidates[0] : currentPath,
          );
        }
      })
      .catch((detectionError) => {
        if (!isCancelled) {
          setTargetDetectionError(getErrorMessage(detectionError));
        }
      })
      .finally(() => {
        if (!isCancelled) {
          setIsDetectingTargets(false);
        }
      });

    return () => {
      isCancelled = true;
    };
  }, [gameID, isOpen, isTargetStep, selectedStrategy]);

  const handleStrategySelect = (strategyType: StrategyType) => {
    if (canReusePreviousSettings && reusePreviousSettings) {
      onReusePreviousSettingsChange?.(false);
    }
    setSelectedStrategyType(strategyType);
    setTargetRelativePath(strategyType === StrategyType.StrategyTypeUnrealPak ? '' : '.');
    setTargetCandidates([]);
    setTargetDetectionWarnings([]);
    setTargetDetectionError(null);
    setPreview(null);
    setPreviewError(null);
  };

  const handleTargetRelativePathChange = (nextTargetRelativePath: string) => {
    if (canReusePreviousSettings && reusePreviousSettings) {
      onReusePreviousSettingsChange?.(false);
    }
    setTargetRelativePath(nextTargetRelativePath);
    setPreview(null);
    setPreviewError(null);
  };

  const loadPreview = async () => {
    if (selectedStrategyType === null || targetRelativePath.trim() === '') {
      return;
    }

    setIsPreviewing(true);
    setPreview(null);
    setPreviewError(null);

    try {
      const loadedPreview = await PreviewImportConfiguration({
        GameID: gameID,
        SourceType: sourceType,
        SourcePath: sourcePath,
        StrategyType: selectedStrategyType,
        TargetRelativePath: targetRelativePath,
      });
      setTargetRelativePath(loadedPreview.TargetRelativePath);
      setPreview(loadedPreview);
      setStep('preview');
    } catch (previewLoadError) {
      setPreviewError(getErrorMessage(previewLoadError));
    } finally {
      setIsPreviewing(false);
    }
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (isBusy) {
      return;
    }

    if (isDetailsStep) {
      if (canContinueFromDetails) {
        setStep('strategy');
      }
      return;
    }

    if (isStrategyStep) {
      if (canContinueFromStrategy) {
        setTargetRelativePath(selectedStrategy?.SupportsTargetDetection ? '' : '.');
        setStep('target');
      }
      return;
    }

    if (isTargetStep) {
      await loadPreview();
      return;
    }

    if (isPreviewStep && trimmedName !== '' && selectedStrategyType !== null && preview !== null) {
      await onImport({
        name: trimmedName,
        strategyType: selectedStrategyType,
        targetRelativePath: preview.TargetRelativePath,
        tags,
      });
    }
  };

  const goBack = () => {
    if (isBusy) {
      return;
    }

    if (isStrategyStep) {
      setStep('details');
    }
    if (isTargetStep) {
      setStep('strategy');
    }
    if (isPreviewStep) {
      setStep('target');
    }
  };

  const canImportAnother = isPreviewStep && onImportAnotherAfterCompleteChange !== undefined;

  const footer = (
    <>
      {!isDetailsStep && (
        <button disabled={isBusy} onClick={goBack} type="button">
          Back
        </button>
      )}
      <button disabled={isBusy} onClick={onClose} type="button">
        Cancel
      </button>
      <div className="game-mod-import-wizard-footer-actions">
        {canImportAnother && (
          <label className="dropdown-menu-checkbox-option game-mod-import-wizard-import-another">
            <input
              checked={importAnotherAfterComplete}
              disabled={isBusy}
              onChange={(event) => onImportAnotherAfterCompleteChange?.(event.target.checked)}
              type="checkbox"
            />
            <span className="dropdown-menu-checkbox-control" aria-hidden="true" />
            <span className="dropdown-menu-item-label">Import another</span>
          </label>
        )}
        <button className="button-main" disabled={isNextDisabled} type="submit">
          {isPreviewStep
            ? isBusy
              ? 'Importing...'
              : 'Import Mod'
            : isTargetStep
              ? isPreviewing
                ? 'Previewing...'
                : 'Preview'
              : 'Next'}
        </button>
      </div>
    </>
  );

  return (
    <Modal
      abovePanel={queueIsland}
      bodyClassName="game-mod-import-wizard-body"
      closeTitle="Close import wizard"
      description="Decide how to import this mod into your game"
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-mod-import-wizard-title"
      onClose={onClose}
      onSubmit={handleSubmit}
      panelClassName="game-mod-import-wizard-panel"
      size="lg"
      title="Import Mod"
      footer={footer}
    >
      {canReusePreviousSettings && (
        <div className="game-mod-import-wizard-reuse-box">
          <label className="dropdown-menu-checkbox-option game-mod-import-wizard-reuse">
            <input
              checked={reusePreviousSettings}
              disabled={isBusy}
              onChange={(event) => onReusePreviousSettingsChange?.(event.target.checked)}
              type="checkbox"
            />
            <span className="dropdown-menu-checkbox-control" aria-hidden="true" />
            <span className="dropdown-menu-item-label">
              Reuse strategy and target from previous import
            </span>
          </label>
        </div>
      )}

      <ol className="game-mod-import-wizard-steps" aria-label="Import steps">
        {stepOrder.map((stepName, index) => (
          <li
            className={
              index < currentStepIndex
                ? 'game-mod-import-wizard-step game-mod-import-wizard-step-complete'
                : stepName === step
                  ? 'game-mod-import-wizard-step game-mod-import-wizard-step-active'
                  : 'game-mod-import-wizard-step'
            }
            key={stepName}
          >
            {index + 1}. {stepLabels[stepName]}
          </li>
        ))}
      </ol>

      <div className="game-mod-import-wizard-scroll">
        <div className="game-mod-import-wizard-content">
          {isDetailsStep && (
            <GameModImportWizardDetailsStep
              isBusy={isBusy}
              availableTags={availableTags}
              name={name}
              onNameChange={setName}
              onTagsChange={setTags}
              sourceLabel={sourceLabel}
              sourcePath={sourcePath}
              tags={tags}
              targetPath={targetPath}
            />
          )}

          {isStrategyStep && (
            <GameModImportWizardStrategyStep
              isBusy={isBusy}
              isLoadingStrategies={isLoadingStrategies}
              onStrategySelect={handleStrategySelect}
              selectedStrategyType={selectedStrategyType}
              strategies={strategies}
              strategyLoadError={strategyLoadError}
              suggestedStrategyType={suggestedStrategyType}
            />
          )}

          {isTargetStep && (
            <GameModImportWizardTargetStep
              candidates={targetCandidates}
              detectionError={targetDetectionError}
              detectionWarnings={targetDetectionWarnings}
              isBusy={isBusy || isPreviewing || isDetectingTargets}
              isDetecting={isDetectingTargets}
              onTargetRelativePathChange={handleTargetRelativePathChange}
              targetRelativePath={targetRelativePath}
            />
          )}

          {isPreviewStep && preview !== null && (
            <GameModImportWizardPreviewStep
              name={trimmedName}
              preview={preview}
              selectedStrategy={selectedStrategy}
              sourceLabel={sourceLabel}
              sourcePath={sourcePath}
              tags={tags}
            />
          )}

          {previewError !== null && <p className="game-mod-import-wizard-error">{previewError}</p>}
          {error !== null && <p className="game-mod-import-wizard-error">{error}</p>}
        </div>
      </div>
    </Modal>
  );
};
