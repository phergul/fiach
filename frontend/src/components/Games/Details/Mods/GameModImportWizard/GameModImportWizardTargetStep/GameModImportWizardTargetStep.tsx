import './GameModImportWizardTargetStep.scss';

interface GameModImportWizardTargetStepProps {
  isBusy: boolean;
  onTargetRelativePathChange: (targetRelativePath: string) => void;
  targetRelativePath: string;
}

export const GameModImportWizardTargetStep = ({
  isBusy,
  onTargetRelativePathChange,
  targetRelativePath,
}: GameModImportWizardTargetStepProps) => {
  return (
    <label className="game-mod-import-wizard-target-step">
      <span className="game-mod-import-wizard-target-step-label">Game-relative target path</span>
      <input
        className="game-mod-import-wizard-target-step-input"
        disabled={isBusy}
        onChange={(event) => onTargetRelativePathChange(event.target.value)}
        placeholder="Data"
        type="text"
        value={targetRelativePath}
      />
      <span className="game-mod-import-wizard-target-step-help">
        Use . for the game root.
      </span>
    </label>
  );
};
