import { Check, Copy, Pencil, Trash2, X } from 'lucide-react';

import type { ModProfile } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './GameProfilesListItem.scss';

interface GameProfilesListItemProps {
  editingProfileName: string;
  isBusy: boolean;
  isEditing: boolean;
  isSelected: boolean;
  enabledModCount: number;
  modCount: number;
  pendingAction: string | null;
  profile: ModProfile;
  onCancelRename: () => void;
  onDeleteProfile: (profile: ModProfile) => void;
  onEditingProfileNameChange: (name: string) => void;
  onRenameProfile: (profileID: number) => void;
  onSelectProfile: (profileID: number) => void;
  onStartRename: (profile: ModProfile) => void;
  onDuplicateProfile: (profile: ModProfile) => void;
}

const formatProfileEditedAt = (updatedAt: string) => {
  if (updatedAt.trim() === '') {
    return 'Edited time unknown';
  }

  const normalizedUpdatedAt = updatedAt.includes('T')
    ? updatedAt
    : `${updatedAt.replace(' ', 'T')}Z`;
  const date = new Date(normalizedUpdatedAt);
  if (Number.isNaN(date.getTime())) {
    return 'Edited time unknown';
  }

  return `Edited ${new Intl.DateTimeFormat(undefined, {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(date)}`;
};

export const GameProfilesListItem = ({
  editingProfileName,
  enabledModCount,
  isBusy,
  isEditing,
  isSelected,
  modCount,
  pendingAction,
  profile,
  onCancelRename,
  onDeleteProfile,
  onEditingProfileNameChange,
  onRenameProfile,
  onSelectProfile,
  onStartRename,
  onDuplicateProfile,
}: GameProfilesListItemProps) => {
  const assignedSummary = `${modCount} ${modCount === 1 ? 'mod' : 'mods'} assigned`;
  const enabledSummary = `${enabledModCount} ${enabledModCount === 1 ? 'mod' : 'mods'} enabled`;
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
              <span className="game-profiles-list-item-meta-part">
                {assignedSummary}
                <span className="game-profiles-list-item-meta-separator" aria-hidden="true">·</span>
                {enabledSummary}
              </span>
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
            <button
              className="game-profiles-list-item-icon-button"
              disabled={isBusy}
              onClick={(event) => {
                event.stopPropagation();
                onDuplicateProfile(profile);
              }}
              title="Duplicate profile"
              type="button"
            >
              <Copy className="game-profiles-list-item-icon" aria-hidden="true" />
            </button>
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
