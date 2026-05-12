import { Check, Pencil, Power, Trash2, X } from 'lucide-react';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import { formatProfileEditedAt } from '../profileFormatting';

import './GameProfilesListItem.scss';

interface GameProfilesListItemProps {
  editingProfileName: string;
  isBusy: boolean;
  isEditing: boolean;
  pendingAction: string | null;
  profile: ModProfile;
  onActivateProfile: (profileID: number) => void;
  onCancelRename: () => void;
  onDeleteProfile: (profile: ModProfile) => void;
  onEditingProfileNameChange: (name: string) => void;
  onRenameProfile: (profileID: number) => void;
  onStartRename: (profile: ModProfile) => void;
}

export const GameProfilesListItem = ({
  editingProfileName,
  isBusy,
  isEditing,
  pendingAction,
  profile,
  onActivateProfile,
  onCancelRename,
  onDeleteProfile,
  onEditingProfileNameChange,
  onRenameProfile,
  onStartRename,
}: GameProfilesListItemProps) => {
  return (
    <li className="game-profiles-list-item">
      <div className="game-profiles-list-item-main">
        {isEditing ? (
          <input
            className="game-profiles-list-item-input"
            disabled={pendingAction === 'rename'}
            onChange={(event) => onEditingProfileNameChange(event.target.value)}
            type="text"
            value={editingProfileName}
            aria-label={`Rename ${profile.Name}`}
          />
        ) : (
          <div className="game-profiles-list-item-copy">
            <div className="game-profiles-list-item-title">
              <span className="game-profiles-list-item-name">{profile.Name}</span>
            </div>
            <span className="game-profiles-list-item-meta">
              0 mods applied · {formatProfileEditedAt(profile.UpdatedAt)}
            </span>
          </div>
        )}
      </div>

      <div className="game-profiles-list-item-actions">
        {isEditing ? (
          <>
            <button
              className="game-profiles-list-item-icon-button"
              disabled={pendingAction === 'rename'}
              onClick={() => onRenameProfile(profile.ID)}
              title="Save profile name"
              type="button"
            >
              <Check className="game-profiles-list-item-icon" aria-hidden="true" />
            </button>
            <button
              className="game-profiles-list-item-icon-button"
              disabled={pendingAction === 'rename'}
              onClick={onCancelRename}
              title="Cancel rename"
              type="button"
            >
              <X className="game-profiles-list-item-icon" aria-hidden="true" />
            </button>
          </>
        ) : (
          <>
            {!profile.IsActive && (
              <button
                className="game-profiles-list-item-button"
                disabled={isBusy}
                onClick={() => onActivateProfile(profile.ID)}
                type="button"
              >
                <Power className="game-profiles-list-item-button-icon" aria-hidden="true" />
                <span>Activate</span>
              </button>
            )}
            <button
              className="game-profiles-list-item-icon-button"
              disabled={isBusy}
              onClick={() => onStartRename(profile)}
              title="Rename profile"
              type="button"
            >
              <Pencil className="game-profiles-list-item-icon" aria-hidden="true" />
            </button>
            <button
              className="game-profiles-list-item-icon-button game-profiles-list-item-icon-button-danger"
              disabled={isBusy}
              onClick={() => onDeleteProfile(profile)}
              title="Delete profile"
              type="button"
            >
              <Trash2 className="game-profiles-list-item-icon" aria-hidden="true" />
            </button>
          </>
        )}
      </div>
    </li>
  );
};
