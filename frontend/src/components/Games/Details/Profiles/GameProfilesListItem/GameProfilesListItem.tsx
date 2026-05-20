import { Check, Pencil, Power, Trash2, X } from 'lucide-react';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import { formatProfileEditedAt } from '../profileFormatting';

import './GameProfilesListItem.scss';

interface GameProfilesListItemProps {
  editingProfileName: string;
  isBusy: boolean;
  isEditing: boolean;
  isSelected: boolean;
  modCount: number;
  pendingAction: string | null;
  profile: ModProfile;
  onActivateProfile: (profileID: number) => void;
  onCancelRename: () => void;
  onDeleteProfile: (profile: ModProfile) => void;
  onEditingProfileNameChange: (name: string) => void;
  onRenameProfile: (profileID: number) => void;
  onSelectProfile: (profileID: number) => void;
  onStartRename: (profile: ModProfile) => void;
}

export const GameProfilesListItem = ({
  editingProfileName,
  isBusy,
  isEditing,
  isSelected,
  modCount,
  pendingAction,
  profile,
  onActivateProfile,
  onCancelRename,
  onDeleteProfile,
  onEditingProfileNameChange,
  onRenameProfile,
  onSelectProfile,
  onStartRename,
}: GameProfilesListItemProps) => {
  const modSummary = `${modCount} ${modCount === 1 ? 'mod' : 'mods'} assigned`;
  const editedSummary = formatProfileEditedAt(profile.UpdatedAt);

  return (
    <li
      className={isSelected ? 'game-profiles-list-item game-profiles-list-item-selected' : 'game-profiles-list-item'}
      onClick={() => {
        if (!isEditing) {
          onSelectProfile(profile.ID);
        }
      }}
    >
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
          <button
            className="game-profiles-list-item-selector"
            onClick={(event) => {
              event.stopPropagation();
              onSelectProfile(profile.ID);
            }}
            type="button"
            aria-current={isSelected ? 'true' : undefined}
          >
            <span className="game-profiles-list-item-title">
              <span className="game-profiles-list-item-name">{profile.Name}</span>
            </span>
            <span className="game-profiles-list-item-meta">
              <span className="game-profiles-list-item-meta-part">{modSummary}</span>
              <span className="game-profiles-list-item-meta-part">{editedSummary}</span>
            </span>
          </button>
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
            {profile.IsActive ? (
              <span className="game-profiles-list-item-action-spacer" aria-hidden="true" />
            ) : (
              <button
                className="game-profiles-list-item-icon-button"
                disabled={isBusy}
                onClick={(event) => {
                  event.stopPropagation();
                  onActivateProfile(profile.ID);
                }}
                title="Activate profile"
                type="button"
              >
                <Power className="game-profiles-list-item-icon" aria-hidden="true" />
              </button>
            )}
            <button
              className="game-profiles-list-item-icon-button"
              disabled={isBusy}
              onClick={(event) => {
                event.stopPropagation();
                onStartRename(profile);
              }}
              title="Rename profile"
              type="button"
            >
              <Pencil className="game-profiles-list-item-icon" aria-hidden="true" />
            </button>
            <button
              className="game-profiles-list-item-icon-button game-profiles-list-item-icon-button-danger"
              disabled={isBusy}
              onClick={(event) => {
                event.stopPropagation();
                onDeleteProfile(profile);
              }}
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
