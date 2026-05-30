import { FormEvent, useEffect, useState } from 'react';

import type { Preview, StrategyDescriptor, StrategyType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import type { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ListImportStrategies, PreviewImportConfiguration } from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import { Modal } from '@components/Common/Modal/Modal';
import { GameModImportWizardDetailsStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardDetailsStep/GameModImportWizardDetailsStep';
import { GameModImportWizardPreviewStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardPreviewStep/GameModImportWizardPreviewStep';
import { GameModImportWizardStrategyStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardStrategyStep/GameModImportWizardStrategyStep';
import { GameModImportWizardTargetStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardTargetStep/GameModImportWizardTargetStep';
import { getErrorMessage } from '@utils';

import './GameModImportWizard.scss';

type GameModImportWizardStep = 'details' | 'strategy' | 'target' | 'preview';

interface GameModImportWizardSubmitInput {
  name: string;
  strategyType: StrategyType;
  targetRelativePath: string;
}

interface GameModImportWizardProps {
  error: string | null;
  gameID: number;
  initialName: string;
  isBusy: boolean;
  isOpen: boolean;
  onClose: () => void;
  onImport: (input: GameModImportWizardSubmitInput) => Promise<void> | void;
  sourceLabel: string;
  sourcePath: string;
  sourceType: ModSourceType;
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
  gameID,
  initialName,
  isBusy,
  isOpen,
  onClose,
  onImport,
  sourceLabel,
  sourcePath,
  sourceType,
  targetPath,
}: GameModImportWizardProps) => {
  const [step, setStep] = useState<GameModImportWizardStep>('details');
  const [name, setName] = useState(initialName);
  const [selectedStrategyType, setSelectedStrategyType] = useState<StrategyType | null>(null);
  const [targetRelativePath, setTargetRelativePath] = useState('.');
  const [preview, setPreview] = useState<Preview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [isPreviewing, setIsPreviewing] = useState(false);
  const [strategies, setStrategies] = useState<StrategyDescriptor[]>([]);
  const [strategyLoadError, setStrategyLoadError] = useState<string | null>(null);
  const [isLoadingStrategies, setIsLoadingStrategies] = useState(false);
  const trimmedName = name.trim();
  const selectedStrategy = strategies.find((strategy) => strategy.Type === selectedStrategyType);
  const isDetailsStep = step === 'details';
  const isStrategyStep = step === 'strategy';
  const isTargetStep = step === 'target';
  const isPreviewStep = step === 'preview';
  const canContinueFromDetails = trimmedName !== '';
  const canContinueFromStrategy = selectedStrategyType !== null && strategyLoadError === null;
  const canPreviewTarget = selectedStrategyType !== null && targetRelativePath.trim() !== '';
  const isNextDisabled = isBusy ||
    isPreviewing ||
    (isDetailsStep && !canContinueFromDetails) ||
    (isStrategyStep && !canContinueFromStrategy) ||
    (isTargetStep && !canPreviewTarget) ||
    (isPreviewStep && preview === null);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    let isCancelled = false;
    setStep('details');
    setName(initialName);
    setSelectedStrategyType(null);
    setTargetRelativePath('.');
    setPreview(null);
    setPreviewError(null);
    setIsPreviewing(false);
    setStrategyLoadError(null);
    setStrategies([]);
    setIsLoadingStrategies(true);

    ListImportStrategies()
      .then((loadedStrategies) => {
        if (isCancelled) {
          return;
        }

        setStrategies(loadedStrategies);
        if (loadedStrategies.length === 1) {
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
  }, [initialName, isOpen]);

  const handleTargetRelativePathChange = (nextTargetRelativePath: string) => {
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

  const footer = (
    <>
      {!isDetailsStep && (
        <button
          className="game-mod-import-wizard-secondary-button"
          disabled={isBusy}
          onClick={goBack}
          type="button"
        >
          Back
        </button>
      )}
      <button
        className="game-mod-import-wizard-primary-button"
        disabled={isNextDisabled}
        type="submit"
      >
        {isPreviewStep ? (isBusy ? 'Importing...' : 'Import Mod') : isTargetStep ? (isPreviewing ? 'Previewing...' : 'Preview') : 'Next'}
      </button>
      <button
        className="game-mod-import-wizard-secondary-button"
        disabled={isBusy}
        onClick={onClose}
        type="button"
      >
        Cancel
      </button>
    </>
  );

  return (
    <Modal
      bodyClassName="game-mod-import-wizard-body"
      closeTitle="Close import wizard"
      description="Decide how to import this mod into your game"
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-mod-import-wizard-title"
      onClose={onClose}
      onSubmit={handleSubmit}
      size="lg"
      title="Import Mod"
      footer={footer}
    >
      <ol className="game-mod-import-wizard-steps" aria-label="Import steps">
        {stepOrder.map((stepName, index) => (
          <li
            className={stepName === step
              ? 'game-mod-import-wizard-step game-mod-import-wizard-step-active'
              : 'game-mod-import-wizard-step'}
            key={stepName}
          >
            <span className="game-mod-import-wizard-step-number">{index + 1}</span>
            <span className="game-mod-import-wizard-step-label">{stepLabels[stepName]}</span>
          </li>
        ))}
      </ol>

      {isDetailsStep && (
        <GameModImportWizardDetailsStep
          isBusy={isBusy}
          name={name}
          onNameChange={setName}
          sourceLabel={sourceLabel}
          sourcePath={sourcePath}
          targetPath={targetPath}
        />
      )}

      {isStrategyStep && (
        <GameModImportWizardStrategyStep
          isBusy={isBusy}
          isLoadingStrategies={isLoadingStrategies}
          onStrategySelect={setSelectedStrategyType}
          selectedStrategyType={selectedStrategyType}
          strategies={strategies}
          strategyLoadError={strategyLoadError}
        />
      )}

      {isTargetStep && (
        <GameModImportWizardTargetStep
          isBusy={isBusy || isPreviewing}
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
        />
      )}

      {previewError !== null && <p className="game-mod-import-wizard-error">{previewError}</p>}
      {error !== null && <p className="game-mod-import-wizard-error">{error}</p>}
    </Modal>
  );
};
