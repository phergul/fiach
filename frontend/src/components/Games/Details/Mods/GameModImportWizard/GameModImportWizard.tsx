import { FormEvent, useEffect, useState } from 'react';

import type { StrategyDescriptor, StrategyType } from '@bindings/github.com/phergul/mod-manager/internal/installconfig/models';
import { ListImportStrategies } from '@bindings/github.com/phergul/mod-manager/internal/services/modservice';
import { Modal } from '@components/Common/Modal/Modal';
import { GameModImportWizardDetailsStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardDetailsStep/GameModImportWizardDetailsStep';
import { GameModImportWizardReviewStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardReviewStep/GameModImportWizardReviewStep';
import { GameModImportWizardStrategyStep } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizardStrategyStep/GameModImportWizardStrategyStep';
import { getErrorMessage } from '@utils';

import './GameModImportWizard.scss';

type GameModImportWizardStep = 'details' | 'strategy' | 'review';

interface GameModImportWizardSubmitInput {
  name: string;
  strategyType: StrategyType;
}

interface GameModImportWizardProps {
  error: string | null;
  initialName: string;
  isBusy: boolean;
  isOpen: boolean;
  onClose: () => void;
  onImport: (input: GameModImportWizardSubmitInput) => Promise<void> | void;
  sourceLabel: string;
  sourcePath: string;
  targetPath: string;
}

const stepLabels: Record<GameModImportWizardStep, string> = {
  details: 'Details',
  strategy: 'Strategy',
  review: 'Review',
};

const stepOrder: GameModImportWizardStep[] = ['details', 'strategy', 'review'];

export const GameModImportWizard = ({
  error,
  initialName,
  isBusy,
  isOpen,
  onClose,
  onImport,
  sourceLabel,
  sourcePath,
  targetPath,
}: GameModImportWizardProps) => {
  const [step, setStep] = useState<GameModImportWizardStep>('details');
  const [name, setName] = useState(initialName);
  const [selectedStrategyType, setSelectedStrategyType] = useState<StrategyType | null>(null);
  const [strategies, setStrategies] = useState<StrategyDescriptor[]>([]);
  const [strategyLoadError, setStrategyLoadError] = useState<string | null>(null);
  const [isLoadingStrategies, setIsLoadingStrategies] = useState(false);
  const trimmedName = name.trim();
  const selectedStrategy = strategies.find((strategy) => strategy.type === selectedStrategyType);
  const isDetailsStep = step === 'details';
  const isStrategyStep = step === 'strategy';
  const isReviewStep = step === 'review';
  const canContinueFromDetails = trimmedName !== '';
  const canContinueFromStrategy = selectedStrategyType !== null && strategyLoadError === null;
  const isNextDisabled = isBusy ||
    (isDetailsStep && !canContinueFromDetails) ||
    (isStrategyStep && !canContinueFromStrategy);

  useEffect(() => {
    if (!isOpen) {
      return;
    }

    let isCancelled = false;
    setStep('details');
    setName(initialName);
    setSelectedStrategyType(null);
    setStrategyLoadError(null);
    setStrategies([]);
    setIsLoadingStrategies(true);

    ListImportStrategies()
      .then((loadedStrategies) => {
        if (isCancelled) {
          return;
        }

        setStrategies(loadedStrategies);
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
        setStep('review');
      }
      return;
    }

    if (isReviewStep && trimmedName !== '' && selectedStrategyType !== null) {
      await onImport({
        name: trimmedName,
        strategyType: selectedStrategyType,
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
    if (isReviewStep) {
      setStep('strategy');
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
        disabled={isNextDisabled || (isReviewStep && selectedStrategyType === null)}
        type="submit"
      >
        {isReviewStep ? (isBusy ? 'Importing...' : 'Import Mod') : 'Next'}
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
      description="Choose how this mod should be prepared for later profile planning."
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

      {isReviewStep && (
        <GameModImportWizardReviewStep
          name={trimmedName}
          selectedStrategy={selectedStrategy}
          sourceLabel={sourceLabel}
          sourcePath={sourcePath}
          targetPath={targetPath}
        />
      )}

      {error !== null && <p className="game-mod-import-wizard-error">{error}</p>}
    </Modal>
  );
};
