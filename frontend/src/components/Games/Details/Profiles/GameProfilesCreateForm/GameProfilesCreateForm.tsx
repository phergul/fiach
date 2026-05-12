import { FormEvent } from 'react';

import { ChevronDown, ChevronUp, Plus } from 'lucide-react';

import './GameProfilesCreateForm.scss';

interface GameProfilesCreateFormProps {
  isCreateOpen: boolean;
  newProfileName: string;
  pendingAction: string | null;
  onCancelCreate: () => void;
  onCreateProfile: (event: FormEvent<HTMLFormElement>) => void;
  onNewProfileNameChange: (name: string) => void;
  onToggleCreate: () => void;
}

export const GameProfilesCreateForm = ({
  isCreateOpen,
  newProfileName,
  pendingAction,
  onCancelCreate,
  onCreateProfile,
  onNewProfileNameChange,
  onToggleCreate,
}: GameProfilesCreateFormProps) => {
  return (
    <>
      <button
        className="game-profiles-create-form-toggle"
        onClick={onToggleCreate}
        type="button"
        aria-expanded={isCreateOpen}
      >
        <span className="game-profiles-create-form-toggle-copy">
          <Plus className="game-profiles-create-form-toggle-icon" aria-hidden="true" />
          <span>Create New Profile</span>
        </span>
        {isCreateOpen ? (
          <ChevronUp className="game-profiles-create-form-toggle-icon" aria-hidden="true" />
        ) : (
          <ChevronDown className="game-profiles-create-form-toggle-icon" aria-hidden="true" />
        )}
      </button>

      {isCreateOpen && (
        <form className="game-profiles-create-form" onSubmit={onCreateProfile}>
          <label className="game-profiles-create-form-label" htmlFor="game-profile-name">
            Profile Name
          </label>
          <input
            className="game-profiles-create-form-input"
            disabled={pendingAction === 'create'}
            id="game-profile-name"
            onChange={(event) => onNewProfileNameChange(event.target.value)}
            placeholder="Enter name for new profile"
            type="text"
            value={newProfileName}
          />
          <div className="game-profiles-create-form-actions">
            <button
              className="game-profiles-create-form-button game-profiles-create-form-button-primary"
              disabled={pendingAction === 'create'}
              type="submit"
            >
              Create Profile
            </button>
            <button
              className="game-profiles-create-form-button"
              disabled={pendingAction === 'create'}
              onClick={onCancelCreate}
              type="button"
            >
              Cancel
            </button>
          </div>
        </form>
      )}
    </>
  );
};
