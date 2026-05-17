import { Check, Loader2 } from 'lucide-react';

import type { StrategyDescriptor, StrategyType } from '@bindings/github.com/phergul/mod-manager/internal/installconfig/models';

import './GameModImportWizardStrategyStep.scss';

interface GameModImportWizardStrategyStepProps {
  isBusy: boolean;
  isLoadingStrategies: boolean;
  onStrategySelect: (strategyType: StrategyType) => void;
  selectedStrategyType: StrategyType | null;
  strategies: StrategyDescriptor[];
  strategyLoadError: string | null;
}

export const GameModImportWizardStrategyStep = ({
  isBusy,
  isLoadingStrategies,
  onStrategySelect,
  selectedStrategyType,
  strategies,
  strategyLoadError,
}: GameModImportWizardStrategyStepProps) => {
  if (isLoadingStrategies) {
    return (
      <p className="game-mod-import-wizard-strategy-step-muted">
        <Loader2 className="game-mod-import-wizard-strategy-step-inline-icon" aria-hidden="true" />
        Loading strategies...
      </p>
    );
  }

  if (strategyLoadError !== null) {
    return <p className="game-mod-import-wizard-strategy-step-error">{strategyLoadError}</p>;
  }

  if (strategies.length === 0) {
    return <p className="game-mod-import-wizard-strategy-step-error">No import strategies are available.</p>;
  }

  return (
    <fieldset className="game-mod-import-wizard-strategy-step">
      <legend className="game-mod-import-wizard-strategy-step-label">Install strategy</legend>
      {strategies.map((strategy) => {
        const isSelected = selectedStrategyType === strategy.Type;

        return (
          <button
            className={isSelected
              ? 'game-mod-import-wizard-strategy-step-option game-mod-import-wizard-strategy-step-option-selected'
              : 'game-mod-import-wizard-strategy-step-option'}
            disabled={isBusy}
            key={strategy.Type}
            onClick={() => onStrategySelect(strategy.Type)}
            type="button"
            aria-pressed={isSelected}
          >
            <span className="game-mod-import-wizard-strategy-step-copy">
              <span className="game-mod-import-wizard-strategy-step-option-label">{strategy.Label}</span>
              <span className="game-mod-import-wizard-strategy-step-description">{strategy.Description}</span>
            </span>
            {isSelected && <Check className="game-mod-import-wizard-strategy-step-option-icon" aria-hidden="true" />}
          </button>
        );
      })}
    </fieldset>
  );
};
