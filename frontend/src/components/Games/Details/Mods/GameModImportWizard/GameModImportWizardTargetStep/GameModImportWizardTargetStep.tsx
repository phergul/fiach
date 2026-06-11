import { Loader2 } from 'lucide-react';

import './GameModImportWizardTargetStep.scss';

interface GameModImportWizardTargetStepProps {
  candidates: string[];
  detectionError: string | null;
  detectionWarnings: string[];
  isBusy: boolean;
  isDetecting: boolean;
  onTargetRelativePathChange: (targetRelativePath: string) => void;
  targetRelativePath: string;
}

export const GameModImportWizardTargetStep = ({
  candidates,
  detectionError,
  detectionWarnings,
  isBusy,
  isDetecting,
  onTargetRelativePathChange,
  targetRelativePath,
}: GameModImportWizardTargetStepProps) => {
  return (
    <div className="game-mod-import-wizard-target-step">
      {isDetecting && (
        <p className="game-mod-import-wizard-target-step-status">
          <Loader2 className="game-mod-import-wizard-target-step-status-icon" aria-hidden="true" />
          Detecting install targets...
        </p>
      )}

      {!isDetecting && candidates.length > 0 && (
        <fieldset className="game-mod-import-wizard-target-step-candidates">
          <legend className="game-mod-import-wizard-target-step-label">Detected targets</legend>
          {candidates.map((candidate) => (
            <button
              aria-pressed={targetRelativePath === candidate}
              className={targetRelativePath === candidate
                ? 'game-mod-import-wizard-target-step-candidate game-mod-import-wizard-target-step-candidate-selected'
                : 'game-mod-import-wizard-target-step-candidate'}
              disabled={isBusy}
              key={candidate}
              onClick={() => onTargetRelativePathChange(candidate)}
              type="button"
            >
              {candidate}
            </button>
          ))}
        </fieldset>
      )}

      <label className="game-mod-import-wizard-target-step-field">
        <span className="game-mod-import-wizard-target-step-label">Game-relative target path</span>
        <input
          className="game-mod-import-wizard-target-step-input"
          disabled={isBusy}
          onChange={(event) => onTargetRelativePathChange(event.target.value)}
          placeholder="Project/Content/Paks/~mods"
          type="text"
          value={targetRelativePath}
        />
        <span className="game-mod-import-wizard-target-step-help">
          Use a detected target or enter a path relative to the game folder.
        </span>
      </label>

      {detectionWarnings.map((warning) => (
        <p className="game-mod-import-wizard-target-step-warning" key={warning}>{warning}</p>
      ))}
      {detectionError !== null && (
        <p className="game-mod-import-wizard-target-step-error">{detectionError}</p>
      )}
    </div>
  );
};
