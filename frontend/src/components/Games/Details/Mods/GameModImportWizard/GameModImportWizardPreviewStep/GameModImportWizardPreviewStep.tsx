import type {
  Preview,
  StrategyDescriptor,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatModMetadataBytes } from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';
import type { ModTagSelection } from '@components/Games/Details/Mods/ModTags/ModTagEditor/ModTagEditor';
import { ModTagChip } from '@components/Games/Details/Mods/ModTags/ModTagChip/ModTagChip';

import './GameModImportWizardPreviewStep.scss';

interface GameModImportWizardPreviewStepProps {
  name: string;
  preview: Preview;
  selectedStrategy: StrategyDescriptor | undefined;
  sourceLabel: string;
  sourcePath: string;
  tags: ModTagSelection[];
}

export const GameModImportWizardPreviewStep = ({
  name,
  preview,
  selectedStrategy,
  sourceLabel,
  sourcePath,
  tags,
}: GameModImportWizardPreviewStepProps) => {
  const extraWarnings = preview.Warnings.filter(
    (warning) => !warning.includes(`first ${preview.Cap}`),
  );

  return (
    <div className="game-mod-import-wizard-preview-step">
      <div className="game-mod-import-wizard-preview-step-summary">
        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Mod name</span>
          <span className="game-mod-import-wizard-preview-step-value">{name}</span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Tags</span>
          <span className="game-mod-import-wizard-preview-step-tags">
            {tags.length === 0 ? (
              <span className="game-mod-import-wizard-preview-step-value">None</span>
            ) : (
              tags.map((tag, index) => (
                <ModTagChip
                  color={tag.Color}
                  key={tag.ID ?? `new-${tag.Name}-${index}`}
                  name={tag.Name}
                />
              ))
            )}
          </span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Install strategy</span>
          <span className="game-mod-import-wizard-preview-step-value">
            {selectedStrategy?.Label ?? 'Not selected'}
          </span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Target path</span>
          <span className="game-mod-import-wizard-preview-step-value">
            {preview.TargetDisplayPath}
          </span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Number of files</span>
          <span className="game-mod-import-wizard-preview-step-value">
            {preview.TotalFileCount}
          </span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Number of folders</span>
          <span className="game-mod-import-wizard-preview-step-value">
            {preview.TotalDirectoryCount}
          </span>
        </div>

        <div className="game-mod-import-wizard-preview-step-row">
          <span className="game-mod-import-wizard-preview-step-label">Size</span>
          <span className="game-mod-import-wizard-preview-step-value">
            {formatModMetadataBytes(preview.TotalSizeBytes)}
          </span>
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
