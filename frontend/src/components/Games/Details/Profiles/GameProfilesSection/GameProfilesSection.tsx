import { FormEvent, useEffect, useMemo, useState } from 'react';

import type { ModProfile } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { useToast } from '@components/Common/Toast/Toast';
import { GameProfileModsPanel } from '@components/Games/Details/Profiles/GameProfileModsPanel/GameProfileModsPanel';
import { GameProfilesAppliedSummary } from '@components/Games/Details/Profiles/GameProfilesAppliedSummary/GameProfilesAppliedSummary';
import { GameProfilesCreateForm } from '@components/Games/Details/Profiles/GameProfilesCreateForm/GameProfilesCreateForm';
import { GameProfilesList } from '@components/Games/Details/Profiles/GameProfilesList/GameProfilesList';
import type { UseAppliedProfileResult, UseGameModsResult, UseGameProfilesResult } from '@hooks';

import './GameProfilesSection.scss';

interface GameProfilesSectionProps {
  appliedProfileManager: UseAppliedProfileResult;
  applyProfilePath: string;
  gameModManager: UseGameModsResult;
  profileManager: UseGameProfilesResult;
  onRestoreVanilla: () => void;
}

export const GameProfilesSection = ({
  appliedProfileManager,
  applyProfilePath,
  gameModManager,
  profileManager,
  onRestoreVanilla,
}: GameProfilesSectionProps) => {
  const { addToast } = useToast();
  const { isLoading: isGameModsLoading, mods: gameMods } = gameModManager;
  const {
    addModsToProfile,
    createProfile,
    deleteProfile,
    isLoading,
    loadError,
    pendingAction,
    profileModsByProfileID,
    profiles,
    removeModFromProfile,
    reorderProfileMods,
    refreshProfiles,
    renameProfile,
    setProfileModEnabled,
  } = profileManager;
  const { appliedProfile } = appliedProfileManager;
  const [newProfileName, setNewProfileName] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editingProfileID, setEditingProfileID] = useState<number | null>(null);
  const [editingProfileName, setEditingProfileName] = useState('');
  const [deleteCandidate, setDeleteCandidate] = useState<ModProfile | null>(null);
  const [selectedProfileID, setSelectedProfileID] = useState<number | null>(null);
  const isBusy = pendingAction !== null || appliedProfileManager.pendingAction !== null;
  const selectedProfile = useMemo(
    () => profiles.find((profile) => profile.ID === selectedProfileID) ?? profiles[0] ?? null,
    [profiles, selectedProfileID],
  );
  const selectedProfileMods = selectedProfile === null ? [] : profileModsByProfileID[selectedProfile.ID] ?? [];

  useEffect(() => {
    if (selectedProfileID !== null && profiles.some((profile) => profile.ID === selectedProfileID)) {
      return;
    }

    setSelectedProfileID(profiles[0]?.ID ?? null);
  }, [profiles, selectedProfileID]);

  const handleCreateProfile = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmedName = newProfileName.trim();
    if (trimmedName === '') {
      addToast({
        message: 'Profile name is required.',
        tone: 'error',
      });
      return;
    }

    try {
      await createProfile(trimmedName);
      setNewProfileName('');
      setIsCreateOpen(false);
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  const startRename = (profile: ModProfile) => {
    setSelectedProfileID(profile.ID);
    setEditingProfileID(profile.ID);
    setEditingProfileName(profile.Name);
  };

  const cancelRename = () => {
    setEditingProfileID(null);
    setEditingProfileName('');
  };

  const handleRenameProfile = async (profileID: number) => {
    const trimmedName = editingProfileName.trim();
    if (trimmedName === '') {
      addToast({
        message: 'Profile name is required.',
        tone: 'error',
      });
      return;
    }

    try {
      await renameProfile(profileID, trimmedName);
      cancelRename();
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  const handleDeleteProfile = async () => {
    if (deleteCandidate === null) {
      return;
    }

    try {
      await deleteProfile(deleteCandidate.ID);
      setDeleteCandidate(null);
      if (selectedProfileID === deleteCandidate.ID) {
        setSelectedProfileID(null);
      }
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  const handleCancelCreate = () => {
    setIsCreateOpen(false);
    setNewProfileName('');
  };

  const refreshAppliedProfileIfChanged = async (profileID: number) => {
    if (appliedProfile?.ProfileID !== profileID) {
      return;
    }

    await appliedProfileManager.refreshAppliedProfile().catch(() => undefined);
  };

  const handleAddModsToProfile = async (profileID: number, modIDs: number[]) => {
    await addModsToProfile(profileID, modIDs);
    await refreshAppliedProfileIfChanged(profileID);
  };

  const handleRemoveModFromProfile = async (profileID: number, modID: number) => {
    try {
      await removeModFromProfile(profileID, modID);
      await refreshAppliedProfileIfChanged(profileID);
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  const handleSetProfileModEnabled = async (profileID: number, modID: number, enabled: boolean) => {
    try {
      await setProfileModEnabled(profileID, modID, enabled);
      await refreshAppliedProfileIfChanged(profileID);
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  const handleReorderProfileMods = async (profileID: number, orderedModIDs: number[]) => {
    try {
      await reorderProfileMods(profileID, orderedModIDs);
      await refreshAppliedProfileIfChanged(profileID);
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  return (
    <section className="game-profiles-section" aria-label="Profiles">
      {loadError !== null && (
        <StateBlock className="game-profiles-section-state" title="Could not load profiles." message={loadError}>
          <button
            className="game-profiles-section-button"
            onClick={refreshProfiles}
            type="button"
          >
            Retry
          </button>
        </StateBlock>
      )}

      {loadError === null && (
        <div className="game-profiles-section-workspace">
          <aside className="game-profiles-section-sidebar" aria-label="Profile list">
            <GameProfilesAppliedSummary
              appliedProfile={appliedProfile}
              isBusy={isBusy}
              onRestoreVanilla={onRestoreVanilla}
            />

            <GameProfilesCreateForm
              isCreateOpen={isCreateOpen}
              newProfileName={newProfileName}
              pendingAction={pendingAction}
              onCancelCreate={handleCancelCreate}
              onCreateProfile={handleCreateProfile}
              onNewProfileNameChange={setNewProfileName}
              onToggleCreate={() => setIsCreateOpen((currentValue) => !currentValue)}
            />

            <GameProfilesList
              editingProfileID={editingProfileID}
              editingProfileName={editingProfileName}
              appliedProfileID={appliedProfile?.ProfileID ?? null}
              isBusy={isBusy}
              isLoading={isLoading}
              pendingAction={pendingAction}
              profileModsByProfileID={profileModsByProfileID}
              profiles={profiles}
              selectedProfileID={selectedProfile?.ID ?? null}
              onCancelRename={cancelRename}
              onDeleteProfile={setDeleteCandidate}
              onEditingProfileNameChange={setEditingProfileName}
              onRenameProfile={handleRenameProfile}
              onSelectProfile={setSelectedProfileID}
              onStartRename={startRename}
            />
          </aside>

          <div className="game-profiles-section-detail">
            <GameProfileModsPanel
              appliedProfile={appliedProfile}
              applyProfilePath={applyProfilePath}
              gameMods={gameMods}
              isBusy={isBusy}
              isGameModsLoading={isGameModsLoading}
              isProfilesLoading={isLoading}
              profile={selectedProfile}
              profileMods={selectedProfileMods}
              onAddModsToProfile={handleAddModsToProfile}
              onRemoveModFromProfile={handleRemoveModFromProfile}
              onReorderProfileMods={handleReorderProfileMods}
              onRestoreVanilla={onRestoreVanilla}
              onSetProfileModEnabled={handleSetProfileModEnabled}
            />
          </div>
        </div>
      )}

      <ConfirmDialog
        confirmLabel="Delete"
        isOpen={deleteCandidate !== null}
        message={
          deleteCandidate === null
            ? ''
            : `Delete "${deleteCandidate.Name}"? This cannot be undone.`
        }
        onCancel={() => setDeleteCandidate(null)}
        onConfirm={handleDeleteProfile}
        title="Delete profile"
      />
    </section>
  );
};
