import { FormEvent, useState } from 'react';

import { Check, Pencil, Plus, Power, PowerOff, Trash2, X } from 'lucide-react';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { useToast } from '@components/Common/Toast/Toast';
import { useGameProfiles } from '@hooks';

import './GameProfilesSection.scss';

interface GameProfilesSectionProps {
  gameID: number;
}

export const GameProfilesSection = ({ gameID }: GameProfilesSectionProps) => {
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
  } = useGameProfiles(gameID);
  const [newProfileName, setNewProfileName] = useState('');
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

    await createProfile(trimmedName);
    setNewProfileName('');
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

    await renameProfile(profileID, trimmedName);
    cancelRename();
  };

  const handleDeleteProfile = async () => {
    if (deleteCandidate === null) {
      return;
    }

    await deleteProfile(deleteCandidate.ID);
    setDeleteCandidate(null);
  };

  return (
    <section className="game-profiles-section" aria-label="Profiles">
      <form className="game-profiles-section-create" onSubmit={handleCreateProfile}>
        <input
          className="game-profiles-section-input"
          disabled={pendingAction === 'create'}
          onChange={(event) => setNewProfileName(event.target.value)}
          placeholder="Profile name"
          type="text"
          value={newProfileName}
          aria-label="Profile name"
        />
        <button
          className="game-profiles-section-button game-profiles-section-button-primary"
          disabled={pendingAction === 'create'}
          type="submit"
        >
          <Plus className="game-profiles-section-button-icon" aria-hidden="true" />
          <span>Create</span>
        </button>
      </form>

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
          <div className="game-profiles-section-active">
            <div className="game-profiles-section-active-copy">
              <span className="game-profiles-section-active-label">Active profile</span>
              <strong className="game-profiles-section-active-name">
                {activeProfile === null ? 'No active profile' : activeProfile.Name}
              </strong>
            </div>
            {activeProfile !== null && (
              <button
                className="game-profiles-section-button"
                disabled={isBusy}
                onClick={clearActiveProfile}
                type="button"
              >
                <PowerOff className="game-profiles-section-button-icon" aria-hidden="true" />
                <span>Clear active</span>
              </button>
            )}
          </div>

          {isLoading && <p className="game-profiles-section-empty">Loading profiles...</p>}

          {!isLoading && profiles.length === 0 && (
            <p className="game-profiles-section-empty">No profiles have been created yet.</p>
          )}

          {!isLoading && profiles.length > 0 && (
            <ul className="game-profiles-section-list">
              {profiles.map((profile) => {
                const isEditing = editingProfileID === profile.ID;

                return (
                  <li
                    className={
                      profile.IsActive
                        ? 'game-profiles-section-row game-profiles-section-row-active'
                        : 'game-profiles-section-row'
                    }
                    key={profile.ID}
                  >
                    <div className="game-profiles-section-row-main">
                      {isEditing ? (
                        <input
                          className="game-profiles-section-input"
                          disabled={pendingAction === 'rename'}
                          onChange={(event) => setEditingProfileName(event.target.value)}
                          type="text"
                          value={editingProfileName}
                          aria-label={`Rename ${profile.Name}`}
                        />
                      ) : (
                        <>
                          <span className="game-profiles-section-row-name">{profile.Name}</span>
                          <span className="game-profiles-section-row-status">
                            {profile.IsActive ? 'Active' : 'Inactive'}
                          </span>
                        </>
                      )}
                    </div>

                    <div className="game-profiles-section-row-actions">
                      {isEditing ? (
                        <>
                          <button
                            className="game-profiles-section-icon-button"
                            disabled={pendingAction === 'rename'}
                            onClick={() => handleRenameProfile(profile.ID)}
                            title="Save profile name"
                            type="button"
                          >
                            <Check className="game-profiles-section-icon" aria-hidden="true" />
                          </button>
                          <button
                            className="game-profiles-section-icon-button"
                            disabled={pendingAction === 'rename'}
                            onClick={cancelRename}
                            title="Cancel rename"
                            type="button"
                          >
                            <X className="game-profiles-section-icon" aria-hidden="true" />
                          </button>
                        </>
                      ) : (
                        <>
                          <button
                            className="game-profiles-section-button"
                            disabled={profile.IsActive || isBusy}
                            onClick={() => activateProfile(profile.ID)}
                            type="button"
                          >
                            <Power className="game-profiles-section-button-icon" aria-hidden="true" />
                            <span>Activate</span>
                          </button>
                          <button
                            className="game-profiles-section-icon-button"
                            disabled={isBusy}
                            onClick={() => startRename(profile)}
                            title="Rename profile"
                            type="button"
                          >
                            <Pencil className="game-profiles-section-icon" aria-hidden="true" />
                          </button>
                          <button
                            className="game-profiles-section-icon-button game-profiles-section-icon-button-danger"
                            disabled={isBusy}
                            onClick={() => setDeleteCandidate(profile)}
                            title="Delete profile"
                            type="button"
                          >
                            <Trash2 className="game-profiles-section-icon" aria-hidden="true" />
                          </button>
                        </>
                      )}
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
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
