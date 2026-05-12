import { FormEvent, useState } from 'react';

import {
  Check,
  ChevronDown,
  ChevronUp,
  Pencil,
  Plus,
  Power,
  PowerOff,
  Trash2,
  X,
} from 'lucide-react';

import type { ModProfile } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { useToast } from '@components/Common/Toast/Toast';
import type { UseGameProfilesResult } from '@hooks';

import './GameProfilesSection.scss';

interface GameProfilesSectionProps {
  profileManager: UseGameProfilesResult;
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
                  onClick={() => {
                    clearActiveProfile().catch(() => undefined);
                  }}
                  type="button"
                >
                  <PowerOff className="game-profiles-section-button-icon" aria-hidden="true" />
                  <span>Clear active</span>
                </button>
              )}
            </div>

            <button
              className="game-profiles-section-create-toggle"
              onClick={() => setIsCreateOpen((currentValue) => !currentValue)}
              type="button"
              aria-expanded={isCreateOpen}
            >
              <span className="game-profiles-section-create-toggle-copy">
                <Plus className="game-profiles-section-create-toggle-icon" aria-hidden="true" />
                <span>Create New Profile</span>
              </span>
              {isCreateOpen ? (
                <ChevronUp className="game-profiles-section-create-toggle-icon" aria-hidden="true" />
              ) : (
                <ChevronDown className="game-profiles-section-create-toggle-icon" aria-hidden="true" />
              )}
            </button>

            {isCreateOpen && (
              <form className="game-profiles-section-create" onSubmit={handleCreateProfile}>
                <label className="game-profiles-section-label" htmlFor="game-profile-name">
                  Profile Name
                </label>
                <input
                  className="game-profiles-section-input"
                  disabled={pendingAction === 'create'}
                  id="game-profile-name"
                  onChange={(event) => setNewProfileName(event.target.value)}
                  placeholder="Enter name for new profile"
                  type="text"
                  value={newProfileName}
                />
                <div className="game-profiles-section-create-actions">
                  <button
                    className="game-profiles-section-button game-profiles-section-button-primary"
                    disabled={pendingAction === 'create'}
                    type="submit"
                  >
                    Create Profile
                  </button>
                  <button
                    className="game-profiles-section-button"
                    disabled={pendingAction === 'create'}
                    onClick={() => {
                      setIsCreateOpen(false);
                      setNewProfileName('');
                    }}
                    type="button"
                  >
                    Cancel
                  </button>
                </div>
              </form>
            )}
          </div>

          <div className="game-profiles-section-list-shell">
            {isLoading && <p className="game-profiles-section-empty">Loading profiles...</p>}

            {!isLoading && profiles.length === 0 && (
              <p className="game-profiles-section-empty">No profiles have been created yet.</p>
            )}

            {!isLoading && profiles.length > 0 && (
              <ul className="game-profiles-section-list">
                {profiles.map((profile) => {
                  const isEditing = editingProfileID === profile.ID;

                  return (
                    <li className="game-profiles-section-row" key={profile.ID}>
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
                          <div className="game-profiles-section-row-copy">
                            <div className="game-profiles-section-row-title">
                              <span className="game-profiles-section-row-name">{profile.Name}</span>
                            </div>
                            <span className="game-profiles-section-row-meta">
                              0 mods applied · {formatProfileEditedAt(profile.UpdatedAt)}
                            </span>
                          </div>
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
                            {!profile.IsActive && (
                              <button
                                className="game-profiles-section-button"
                                disabled={isBusy}
                                onClick={() => {
                                  activateProfile(profile.ID).catch(() => undefined);
                                }}
                                type="button"
                              >
                                <Power
                                  className="game-profiles-section-button-icon"
                                  aria-hidden="true"
                                />
                                <span>Activate</span>
                              </button>
                            )}
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
          </div>
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
