import { ChangeEvent, useEffect, useMemo, useState } from 'react';

import { Pencil, Plus, Power, PowerOff, Trash2 } from 'lucide-react';

import type { Mod, ModProfile, ProfileMod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameProfileModsPanel.scss';

interface GameProfileModsPanelProps {
  gameMods: Mod[];
  isBusy: boolean;
  isGameModsLoading: boolean;
  isProfilesLoading: boolean;
  profile: ModProfile | null;
  profileMods: ProfileMod[];
  onActivateProfile: (profileID: number) => void;
  onAddModToProfile: (profileID: number, modID: number) => void;
  onDeactivateProfile: () => void;
  onDeleteProfile: (profile: ModProfile) => void;
  onRemoveModFromProfile: (profileID: number, modID: number) => void;
  onSetProfileModEnabled: (profileID: number, modID: number, enabled: boolean) => void;
  onStartRename: (profile: ModProfile) => void;
}

export const GameProfileModsPanel = ({
  gameMods,
  isBusy,
  isGameModsLoading,
  isProfilesLoading,
  profile,
  profileMods,
  onActivateProfile,
  onAddModToProfile,
  onDeactivateProfile,
  onDeleteProfile,
  onRemoveModFromProfile,
  onSetProfileModEnabled,
  onStartRename,
}: GameProfileModsPanelProps) => {
  const [selectedModID, setSelectedModID] = useState('');
  const assignedModIDs = useMemo(() => new Set(profileMods.map((profileMod) => profileMod.ModID)), [profileMods]);
  const availableMods = useMemo(
    () => gameMods.filter((mod) => !assignedModIDs.has(mod.ID)),
    [assignedModIDs, gameMods],
  );
  const canAddMod =
    profile !== null &&
    selectedModID !== '' &&
    availableMods.some((mod) => mod.ID === Number(selectedModID)) &&
    !isBusy;

  const handleSelectedModChange = (event: ChangeEvent<HTMLSelectElement>) => {
    setSelectedModID(event.target.value);
  };

  useEffect(() => {
    setSelectedModID('');
  }, [profile?.ID]);

  const handleAddMod = () => {
    if (profile === null || selectedModID === '') {
      return;
    }

    onAddModToProfile(profile.ID, Number(selectedModID));
    setSelectedModID('');
  };

  if (profile === null) {
    return (
      <section className="game-profile-mods-panel" aria-label="Profile mods">
        <p className="game-profile-mods-panel-empty">
          {isProfilesLoading ? 'Loading profile details...' : 'Create a profile to configure mods.'}
        </p>
      </section>
    );
  }

  return (
    <section className="game-profile-mods-panel" aria-label={`${profile.Name} mods`}>
      <div className="game-profile-mods-panel-header">
        <div className="game-profile-mods-panel-title">
          <span className="game-profile-mods-panel-name">
            {profile.Name}
            {profile.IsActive && <span className="game-profile-mods-panel-active">Active</span>}
          </span>
          <span className="game-profile-mods-panel-count">
            {profileMods.length} assigned · {profileMods.filter((profileMod) => profileMod.Enabled).length} enabled
          </span>
        </div>

        <div className="game-profile-mods-panel-header-actions">
          {profile.IsActive ? (
            <button
              className="game-profile-mods-panel-button"
              disabled={isBusy}
              onClick={onDeactivateProfile}
              type="button"
            >
              <PowerOff className="game-profile-mods-panel-icon" aria-hidden="true" />
              <span>Deactivate</span>
            </button>
          ) : (
            <button
              className="game-profile-mods-panel-button"
              disabled={isBusy}
              onClick={() => onActivateProfile(profile.ID)}
              type="button"
            >
              <Power className="game-profile-mods-panel-icon" aria-hidden="true" />
              <span>Activate</span>
            </button>
          )}
          <button
            className="game-profile-mods-panel-icon-button"
            disabled={isBusy}
            onClick={() => onStartRename(profile)}
            title="Rename profile"
            type="button"
          >
            <Pencil className="game-profile-mods-panel-icon" aria-hidden="true" />
          </button>
          <button
            className="game-profile-mods-panel-icon-button game-profile-mods-panel-icon-button-danger"
            disabled={isBusy}
            onClick={() => onDeleteProfile(profile)}
            title="Delete profile"
            type="button"
          >
            <Trash2 className="game-profile-mods-panel-icon" aria-hidden="true" />
          </button>
        </div>
      </div>

      <div className="game-profile-mods-panel-add-row">
        <div className="game-profile-mods-panel-add">
          <select
            className="game-profile-mods-panel-select"
            disabled={isBusy || isGameModsLoading || availableMods.length === 0}
            onChange={handleSelectedModChange}
            value={selectedModID}
            aria-label="Add imported mod to profile"
          >
            <option value="">
              {availableMods.length === 0 ? 'No available mods' : 'Choose imported mod'}
            </option>
            {availableMods.map((mod) => (
              <option key={mod.ID} value={mod.ID}>
                {mod.Name}
              </option>
            ))}
          </select>
          <button
            className="game-profile-mods-panel-add-button"
            disabled={!canAddMod}
            onClick={handleAddMod}
            title="Add mod to profile"
            type="button"
          >
            <Plus className="game-profile-mods-panel-icon" aria-hidden="true" />
            <span>Add</span>
          </button>
        </div>
      </div>

      <div className="game-profile-mods-panel-body">
        {isGameModsLoading && (
          <p className="game-profile-mods-panel-empty">Loading imported mods...</p>
        )}

        {!isGameModsLoading && gameMods.length === 0 && (
          <p className="game-profile-mods-panel-empty">Import mods for this game before assigning them to a profile.</p>
        )}

        {!isGameModsLoading && gameMods.length > 0 && profileMods.length === 0 && (
          <p className="game-profile-mods-panel-empty">No mods are assigned to this profile yet.</p>
        )}

        {profileMods.length > 0 && (
          <ul className="game-profile-mods-panel-list">
            {profileMods.map((profileMod) => (
              <li className="game-profile-mods-panel-list-item" key={profileMod.ModID}>
                <div className="game-profile-mods-panel-mod-copy">
                  <span className="game-profile-mods-panel-mod-name">{profileMod.Name}</span>
                  <span className="game-profile-mods-panel-mod-meta">
                    Load order {profileMod.LoadOrder} · {profileMod.SourcePath}
                  </span>
                </div>

                <div className="game-profile-mods-panel-actions">
                  <label className="game-profile-mods-panel-toggle">
                    <input
                      checked={profileMod.Enabled}
                      disabled={isBusy}
                      onChange={(event) => onSetProfileModEnabled(profile.ID, profileMod.ModID, event.target.checked)}
                      type="checkbox"
                    />
                    <span>{profileMod.Enabled ? 'Enabled' : 'Disabled'}</span>
                  </label>
                  <button
                    className="game-profile-mods-panel-icon-button game-profile-mods-panel-icon-button-danger"
                    disabled={isBusy}
                    onClick={() => onRemoveModFromProfile(profile.ID, profileMod.ModID)}
                    title="Remove mod from profile"
                    type="button"
                  >
                    <Trash2 className="game-profile-mods-panel-icon" aria-hidden="true" />
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
};
