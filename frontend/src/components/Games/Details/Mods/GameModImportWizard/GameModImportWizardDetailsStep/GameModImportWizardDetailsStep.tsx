import './GameModImportWizardDetailsStep.scss';

interface GameModImportWizardDetailsStepProps {
  isBusy: boolean;
  name: string;
  onNameChange: (name: string) => void;
  sourceLabel: string;
  sourcePath: string;
  targetPath: string;
}

export const GameModImportWizardDetailsStep = ({
  isBusy,
  name,
  onNameChange,
  sourceLabel,
  sourcePath,
  targetPath,
}: GameModImportWizardDetailsStepProps) => {
  return (
    <div className="game-mod-import-wizard-details-step">
      <label className="game-mod-import-wizard-details-step-field">
        <span className="game-mod-import-wizard-details-step-label">Mod name</span>
        <input
          className="game-mod-import-wizard-details-step-input"
          disabled={isBusy}
          onChange={(event) => onNameChange(event.target.value)}
          type="text"
          value={name}
        />
      </label>

      <div className="game-mod-import-wizard-details-step-paths">
        <div className="game-mod-import-wizard-details-step-path-row">
          <span className="game-mod-import-wizard-details-step-label">{sourceLabel}</span>
          <span className="game-mod-import-wizard-details-step-path">{sourcePath}</span>
        </div>

        <div className="game-mod-import-wizard-details-step-path-row">
          <span className="game-mod-import-wizard-details-step-label">Managed storage location</span>
          <span className="game-mod-import-wizard-details-step-path">{targetPath}</span>
        </div>
      </div>
    </div>
  );
};
