import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import { GameProfilesListItem } from '../GameProfilesListItem/GameProfilesListItem';

import './GameProfilesList.scss';

interface GameProfilesListProps {
  editingProfileID: number | null;
  editingProfileName: string;
  isBusy: boolean;
  isLoading: boolean;
  pendingAction: string | null;
  profiles: ModProfile[];
  onActivateProfile: (profileID: number) => void;
  onCancelRename: () => void;
  onDeleteProfile: (profile: ModProfile) => void;
  onEditingProfileNameChange: (name: string) => void;
  onRenameProfile: (profileID: number) => void;
  onStartRename: (profile: ModProfile) => void;
}

export const GameProfilesList = ({
  editingProfileID,
  editingProfileName,
  isBusy,
  isLoading,
  pendingAction,
  profiles,
  onActivateProfile,
  onCancelRename,
  onDeleteProfile,
  onEditingProfileNameChange,
  onRenameProfile,
  onStartRename,
}: GameProfilesListProps) => {
  return (
    <div className="game-profiles-list-shell">
      {isLoading && <p className="game-profiles-list-empty">Loading profiles...</p>}

      {!isLoading && profiles.length === 0 && (
        <p className="game-profiles-list-empty">No profiles have been created yet.</p>
      )}

      {!isLoading && profiles.length > 0 && (
        <ul className="game-profiles-list">
          {profiles.map((profile) => (
            <GameProfilesListItem
              editingProfileName={editingProfileName}
              isBusy={isBusy}
              isEditing={editingProfileID === profile.ID}
              key={profile.ID}
              pendingAction={pendingAction}
              profile={profile}
              onActivateProfile={onActivateProfile}
              onCancelRename={onCancelRename}
              onDeleteProfile={onDeleteProfile}
              onEditingProfileNameChange={onEditingProfileNameChange}
              onRenameProfile={onRenameProfile}
              onStartRename={onStartRename}
            />
          ))}
        </ul>
      )}
    </div>
  );
};
