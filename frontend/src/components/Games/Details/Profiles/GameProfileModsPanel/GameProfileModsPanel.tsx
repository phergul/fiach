import { useEffect, useMemo, useState } from 'react';

import { Link } from 'react-router-dom';
import { CheckCircle2, Plus, RotateCcw } from 'lucide-react';

import type { AppliedProfileSummary } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import type { Mod, ModProfile, ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { GameProfileAddModsModal } from '@components/Games/Details/Profiles/GameProfileAddModsModal/GameProfileAddModsModal';
import { GameProfileAssignedModsList } from '@components/Games/Details/Profiles/GameProfileAssignedModsList/GameProfileAssignedModsList';

import './GameProfileModsPanel.scss';

interface GameProfileModsPanelProps {
  appliedProfile: AppliedProfileSummary | null;
  applyProfilePath: string;
  gameMods: Mod[];
  isBusy: boolean;
  isGameModsLoading: boolean;
  isProfilesLoading: boolean;
  profile: ModProfile | null;
  profileMods: ProfileMod[];
  onAddModsToProfile: (profileID: number, modIDs: number[]) => Promise<void> | void;
  onRemoveModFromProfile: (profileID: number, modID: number) => void;
  onReorderProfileMods: (profileID: number, orderedModIDs: number[]) => void;
  onRestoreVanilla: () => void;
  onSetProfileModEnabled: (profileID: number, modID: number, enabled: boolean) => void;
}

export const GameProfileModsPanel = ({
  appliedProfile,
  applyProfilePath,
  gameMods,
  isBusy,
  isGameModsLoading,
  isProfilesLoading,
  profile,
  profileMods,
  onAddModsToProfile,
  onRemoveModFromProfile,
  onReorderProfileMods,
  onRestoreVanilla,
  onSetProfileModEnabled,
}: GameProfileModsPanelProps) => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const assignedModIDs = useMemo(() => new Set(profileMods.map((profileMod) => profileMod.ModID)), [profileMods]);
  const availableMods = useMemo(
    () => gameMods.filter((mod) => !assignedModIDs.has(mod.ID)),
    [assignedModIDs, gameMods],
  );
  const canOpenAddModal = !isBusy && !isGameModsLoading && availableMods.length > 0;
  const isSelectedProfileApplied = profile !== null && appliedProfile?.ProfileID === profile.ID;
  const isAnotherProfileApplied = profile !== null && appliedProfile !== null && !isSelectedProfileApplied;
  const blockedApplyTitle = appliedProfile === null
    ? undefined
    : `${appliedProfile.ProfileName} is applied. Restore vanilla before applying another profile.`;

  useEffect(() => {
    setIsAddModalOpen(false);
  }, [profile?.ID]);

  const handleAddMods = async (modIDs: number[]) => {
    if (profile === null || modIDs.length === 0) {
      return;
    }

    await onAddModsToProfile(profile.ID, modIDs);
  };

  const handleMoveProfileMod = (modID: number, direction: -1 | 1) => {
    if (profile === null) {
      return;
    }

    const currentIndex = profileMods.findIndex((profileMod) => profileMod.ModID === modID);
    const nextIndex = currentIndex + direction;
    if (currentIndex < 0 || nextIndex < 0 || nextIndex >= profileMods.length) {
      return;
    }

    const reorderedProfileMods = [...profileMods];
    [reorderedProfileMods[currentIndex], reorderedProfileMods[nextIndex]] = [
      reorderedProfileMods[nextIndex],
      reorderedProfileMods[currentIndex],
    ];

    onReorderProfileMods(
      profile.ID,
      reorderedProfileMods.map((profileMod) => profileMod.ModID),
    );
  };

  if (profile === null) {
    return (
      <section className="game-profile-mods-panel" aria-label="Profile mods">
        <StateBlock
          className="game-profile-mods-panel-empty"
          message={isProfilesLoading ? 'Loading profile details...' : 'Create a profile to configure mods.'}
        />
      </section>
    );
  }

  return (
    <section className="game-profile-mods-panel" aria-label={`${profile.Name} mods`}>
      <div className="game-profile-mods-panel-body">
        {isGameModsLoading && (
          <StateBlock className="game-profile-mods-panel-empty" message="Loading imported mods..." />
        )}

        {!isGameModsLoading && gameMods.length === 0 && (
          <StateBlock
            className="game-profile-mods-panel-empty"
            message="Import mods for this game before assigning them to a profile."
          />
        )}

        {!isGameModsLoading && gameMods.length > 0 && profileMods.length === 0 && (
          <StateBlock
            className="game-profile-mods-panel-empty game-profile-mods-panel-empty-row"
            message="No mods are assigned to this profile yet."
          />
        )}

        {profileMods.length > 0 && (
          <GameProfileAssignedModsList
            isBusy={isBusy}
            mods={profileMods}
            onMoveMod={handleMoveProfileMod}
            onRemoveMod={(modID) => onRemoveModFromProfile(profile.ID, modID)}
            onSetModEnabled={(modID, enabled) => onSetProfileModEnabled(profile.ID, modID, enabled)}
          />
        )}
      </div>

      <div className="game-profile-mods-panel-footer">
        <button
          className="game-profile-mods-panel-add-button"
          disabled={!canOpenAddModal}
          onClick={() => setIsAddModalOpen(true)}
          type="button"
        >
          <Plus className="game-profile-mods-panel-icon" aria-hidden="true" />
          <span>Add Mods from Library</span>
        </button>

        {isSelectedProfileApplied ? (
          <button
            className="game-profile-mods-panel-restore-button"
            disabled={isBusy}
            onClick={onRestoreVanilla}
            type="button"
          >
            <RotateCcw className="game-profile-mods-panel-icon" aria-hidden="true" />
            <span>Restore Vanilla</span>
          </button>
        ) : isAnotherProfileApplied ? (
          <button
            className="game-profile-mods-panel-apply-button"
            disabled
            title={blockedApplyTitle}
            type="button"
          >
            <CheckCircle2 className="game-profile-mods-panel-icon" aria-hidden="true" />
            <span>Another Profile Applied</span>
          </button>
        ) : (
          <Link
            className={isBusy ? 'game-profile-mods-panel-apply-button game-profile-mods-panel-link-disabled' : 'game-profile-mods-panel-apply-button'}
            to={`${applyProfilePath}/${profile.ID}`}
            onClick={(event) => {
              if (isBusy) {
                event.preventDefault();
              }
            }}
            aria-disabled={isBusy}
          >
            <CheckCircle2 className="game-profile-mods-panel-icon" aria-hidden="true" />
            <span>Apply Profile</span>
          </Link>
        )}
      </div>

      <GameProfileAddModsModal
        availableMods={availableMods}
        isBusy={isBusy}
        isGameModsLoading={isGameModsLoading}
        isOpen={isAddModalOpen}
        profileName={profile.Name}
        onAddMods={handleAddMods}
        onClose={() => setIsAddModalOpen(false)}
      />
    </section>
  );
};
