import type { Preview, StrategyDescriptor } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';

import './GameModImportWizardPreviewStep.scss';

interface GameModImportWizardPreviewStepProps {
  name: string;
  preview: Preview;
  selectedStrategy: StrategyDescriptor | undefined;
  sourceLabel: string;
  sourcePath: string;
}

export const GameModImportWizardPreviewStep = ({
  name,
  preview,
  selectedStrategy,
  sourceLabel,
  sourcePath,
}: GameModImportWizardPreviewStepProps) => {
  const extraWarnings = preview.Warnings.filter((warning) => !warning.includes(`first ${preview.Cap}`));

  return (
    <div className="game-mod-import-wizard-preview-step">
      <div className="game-mod-import-wizard-preview-step-summary">
        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Mod name</span>
          <span className="game-mod-import-wizard-preview-step-value">{name}</span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Install strategy</span>
          <span className="game-mod-import-wizard-preview-step-value">{selectedStrategy?.Label ?? 'Not selected'}</span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Target path</span>
          <span className="game-mod-import-wizard-preview-step-value">{preview.TargetDisplayPath}</span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Managed files</span>
          <span className="game-mod-import-wizard-preview-step-value">{preview.TotalFileCount}</span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">{sourceLabel}</span>
          <span className="game-mod-import-wizard-preview-step-path">{sourcePath}</span>
        </div>
      </div>

      {preview.IsCapped && (
        <p className="game-mod-import-wizard-preview-step-warning">
          Showing first {preview.Cap} of {preview.TotalFileCount} target files.
        </p>
      )}

      {extraWarnings.map((warning) => (
        <p className="game-mod-import-wizard-preview-step-warning" key={warning}>
          {warning}
        </p>
      ))}

      <div className="game-mod-import-wizard-preview-step-file-list" aria-label="Target file paths">
        {preview.TargetFilePaths.map((targetFilePath) => (
          <span className="game-mod-import-wizard-preview-step-file" key={targetFilePath}>
            {targetFilePath}
          </span>
        ))}
      </div>
    </div>
  );
};
