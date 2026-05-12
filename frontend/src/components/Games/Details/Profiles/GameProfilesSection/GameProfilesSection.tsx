import { FormEvent, useState } from 'react';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { useToast } from '@components/Common/Toast/Toast';
import { GameProfilesActiveSummary } from '@components/Games/Details/Profiles/GameProfilesActiveSummary/GameProfilesActiveSummary';
import { GameProfilesCreateForm } from '@components/Games/Details/Profiles/GameProfilesCreateForm/GameProfilesCreateForm';
import { GameProfilesList } from '@components/Games/Details/Profiles/GameProfilesList/GameProfilesList';
import type { UseGameProfilesResult } from '@hooks';

import './GameProfilesSection.scss';

interface GameProfilesSectionProps {
  profileManager: UseGameProfilesResult;
}

export const GameProfilesSection = ({ profileManager }: GameProfilesSectionProps) => {
  const { addToast } = useToast();
  const {
    activeProfile,
    activateProfile,
    clearActiveProfile,
    createProfile,
    deleteProfile,
    isLoading,
    loadError,
    pendingAction,
    profiles,
    refreshProfiles,
    renameProfile,
  } = profileManager;
  const [newProfileName, setNewProfileName] = useState('');
  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [editingProfileID, setEditingProfileID] = useState<number | null>(null);
  const [editingProfileName, setEditingProfileName] = useState('');
  const [deleteCandidate, setDeleteCandidate] = useState<ModProfile | null>(null);
  const isBusy = pendingAction !== null;

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
    } catch {
      // error toast is handled by useGameProfiles
    }
  };

  const handleCancelCreate = () => {
    setIsCreateOpen(false);
    setNewProfileName('');
  };

  const handleActivateProfile = (profileID: number) => {
    activateProfile(profileID).catch(() => undefined);
  };

  const handleClearActiveProfile = () => {
    clearActiveProfile().catch(() => undefined);
  };

  return (
    <section className="game-profiles-section" aria-label="Profiles">
      {loadError !== null && (
        <div className="game-profiles-section-state">
          <p className="game-profiles-section-state-title">Could not load profiles.</p>
          <p className="game-profiles-section-state-message">{loadError}</p>
          <button
            className="game-profiles-section-button"
            onClick={refreshProfiles}
            type="button"
          >
            Retry
          </button>
        </div>
      )}

      {loadError === null && (
        <>
          <div className="game-profiles-section-fixed">
            <GameProfilesActiveSummary
              activeProfile={activeProfile}
              isBusy={isBusy}
              onClearActiveProfile={handleClearActiveProfile}
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
          </div>

          <GameProfilesList
            editingProfileID={editingProfileID}
            editingProfileName={editingProfileName}
            isBusy={isBusy}
            isLoading={isLoading}
            pendingAction={pendingAction}
            profiles={profiles}
            onActivateProfile={handleActivateProfile}
            onCancelRename={cancelRename}
            onDeleteProfile={setDeleteCandidate}
            onEditingProfileNameChange={setEditingProfileName}
            onRenameProfile={handleRenameProfile}
            onStartRename={startRename}
          />
        </>
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
