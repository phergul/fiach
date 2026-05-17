import type { StrategyDescriptor } from '@bindings/github.com/phergul/mod-manager/internal/installconfig/models';

import './GameModImportWizardReviewStep.scss';

interface GameModImportWizardReviewStepProps {
  name: string;
  selectedStrategy: StrategyDescriptor | undefined;
  sourceLabel: string;
  sourcePath: string;
  targetPath: string;
}

export const GameModImportWizardReviewStep = ({
  name,
  selectedStrategy,
  sourceLabel,
  sourcePath,
  targetPath,
}: GameModImportWizardReviewStepProps) => {
  return (
    <div className="game-mod-import-wizard-review-step">
      <div className="game-mod-import-wizard-review-step-row">
        <span className="game-mod-import-wizard-review-step-label">Mod name</span>
        <span className="game-mod-import-wizard-review-step-value">{name}</span>
      </div>

      <div className="game-mod-import-wizard-review-step-row">
        <span className="game-mod-import-wizard-review-step-label">Install strategy</span>
        <span className="game-mod-import-wizard-review-step-value">{selectedStrategy?.label ?? 'Not selected'}</span>
      </div>

      <div className="game-mod-import-wizard-review-step-row">
        <span className="game-mod-import-wizard-review-step-label">{sourceLabel}</span>
        <span className="game-mod-import-wizard-review-step-path">{sourcePath}</span>
      </div>

      <div className="game-mod-import-wizard-review-step-row">
        <span className="game-mod-import-wizard-review-step-label">Managed storage location</span>
        <span className="game-mod-import-wizard-review-step-path">{targetPath}</span>
      </div>
    </div>
  );
};
