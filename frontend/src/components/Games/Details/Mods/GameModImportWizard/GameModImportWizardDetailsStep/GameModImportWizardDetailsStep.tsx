import type { Tag } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ModTagEditor,
  type ModTagSelection,
} from '@components/Games/Details/Mods/ModTags/ModTagEditor/ModTagEditor';

import './GameModImportWizardDetailsStep.scss';

interface GameModImportWizardDetailsStepProps {
  availableTags: Tag[];
  isBusy: boolean;
  name: string;
  onNameChange: (name: string) => void;
  onTagsChange: (tags: ModTagSelection[]) => void;
  sourceLabel: string;
  sourcePath: string;
  targetPath: string;
  tags: ModTagSelection[];
}

export const GameModImportWizardDetailsStep = ({
  availableTags,
  isBusy,
  name,
  onNameChange,
  onTagsChange,
  sourceLabel,
  sourcePath,
  targetPath,
  tags,
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

      <div className="game-mod-import-wizard-details-step-field">
        <span className="game-mod-import-wizard-details-step-label">Tags</span>
        <ModTagEditor
          availableTags={availableTags}
          isBusy={isBusy}
          onChange={onTagsChange}
          selectedTags={tags}
        />
      </div>

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
