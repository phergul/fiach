import { useEffect, useMemo, useState } from 'react';

import { Plus } from 'lucide-react';

import type { Mod, ModProfile, ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { GameProfileAddModsModal } from '@components/Games/Details/Profiles/GameProfileAddModsModal/GameProfileAddModsModal';
import { GameProfileAssignedModsList } from '@components/Games/Details/Profiles/GameProfileAssignedModsList/GameProfileAssignedModsList';

import './GameProfileModsPanel.scss';

interface GameProfileModsPanelProps {
  gameMods: Mod[];
  isBusy: boolean;
  isGameModsLoading: boolean;
  isProfilesLoading: boolean;
  profile: ModProfile | null;
  profileMods: ProfileMod[];
  onAddModsToProfile: (profileID: number, modIDs: number[]) => Promise<void> | void;
  onRemoveModFromProfile: (profileID: number, modID: number) => void;
  onSetProfileModEnabled: (profileID: number, modID: number, enabled: boolean) => void;
}

export const GameProfileModsPanel = ({
  gameMods,
  isBusy,
  isGameModsLoading,
  isProfilesLoading,
  profile,
  profileMods,
  onAddModsToProfile,
  onRemoveModFromProfile,
  onSetProfileModEnabled,
}: GameProfileModsPanelProps) => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const assignedModIDs = useMemo(() => new Set(profileMods.map((profileMod) => profileMod.ModID)), [profileMods]);
  const availableMods = useMemo(
    () => gameMods.filter((mod) => !assignedModIDs.has(mod.ID)),
    [assignedModIDs, gameMods],
  );
  const canOpenAddModal = !isBusy && !isGameModsLoading && availableMods.length > 0;

  useEffect(() => {
    setIsAddModalOpen(false);
  }, [profile?.ID]);

  const handleAddMods = async (modIDs: number[]) => {
    if (profile === null || modIDs.length === 0) {
      return;
    }

    await onAddModsToProfile(profile.ID, modIDs);
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
