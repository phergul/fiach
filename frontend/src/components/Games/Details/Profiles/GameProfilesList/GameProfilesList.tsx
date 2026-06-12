import type { ModProfile, ProfileMod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';

import { GameProfilesListItem } from '../GameProfilesListItem/GameProfilesListItem';

import './GameProfilesList.scss';

interface GameProfilesListProps {
  editingProfileID: number | null;
  editingProfileName: string;
  isBusy: boolean;
  isLoading: boolean;
  pendingAction: string | null;
  profileModsByProfileID: Record<number, ProfileMod[]>;
  profiles: ModProfile[];
  selectedProfileID: number | null;
  onCancelRename: () => void;
  onDeleteProfile: (profile: ModProfile) => void;
  onEditingProfileNameChange: (name: string) => void;
  onRenameProfile: (profileID: number) => void;
  onSelectProfile: (profileID: number) => void;
  onStartRename: (profile: ModProfile) => void;
}

export const GameProfilesList = ({
  editingProfileID,
  editingProfileName,
  isBusy,
  isLoading,
  pendingAction,
  profileModsByProfileID,
  profiles,
  selectedProfileID,
  onCancelRename,
  onDeleteProfile,
  onEditingProfileNameChange,
  onRenameProfile,
  onSelectProfile,
  onStartRename,
}: GameProfilesListProps) => {
  return (
    <div className="game-profiles-list-shell">
      {isLoading && <StateBlock className="game-profiles-list-empty" message="Loading profiles..." />}

      {!isLoading && profiles.length === 0 && (
        <StateBlock className="game-profiles-list-empty" message="No profiles have been created yet." />
      )}

      {!isLoading && profiles.length > 0 && (
        <ul className="game-profiles-list">
          {profiles.map((profile) => (
            <GameProfilesListItem
              editingProfileName={editingProfileName}
              enabledModCount={
                profileModsByProfileID[profile.ID]?.filter((profileMod) => profileMod.Enabled).length ?? 0
              }
              isBusy={isBusy}
              isEditing={editingProfileID === profile.ID}
              isSelected={selectedProfileID === profile.ID}
              key={profile.ID}
              modCount={profileModsByProfileID[profile.ID]?.length ?? 0}
              pendingAction={pendingAction}
              profile={profile}
              onCancelRename={onCancelRename}
              onDeleteProfile={onDeleteProfile}
              onEditingProfileNameChange={onEditingProfileNameChange}
              onRenameProfile={onRenameProfile}
              onSelectProfile={onSelectProfile}
              onStartRename={onStartRename}
            />
          ))}
        </ul>
      )}
    </div>
  );
};
