import { useEffect, useMemo, useRef, useState, type MouseEvent } from 'react';

import { Link, useNavigate } from 'react-router-dom';
import { CheckCircle2, FileSearch, Plus } from 'lucide-react';

import type { AppliedProfileSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import type {
  Mod,
  ModProfile,
  ProfileMod,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { InlineLoading } from '@components/Common/InlineLoading/InlineLoading';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { useToast } from '@components/Common/Toast/Toast';
import { GameProfileAddModsModal } from '@components/Games/Details/Profiles/GameProfileAddModsModal/GameProfileAddModsModal';
import { GameProfileAssignedModsList } from '@components/Games/Details/Profiles/GameProfileAssignedModsList/GameProfileAssignedModsList';
import { GameProfileModsFilter } from '@components/Games/Details/Profiles/GameProfileModsFilter/GameProfileModsFilter';
import { ModTagFilter } from '@components/Games/Details/Mods/ModTags/ModTagFilter/ModTagFilter';
import {
  fetchAppliedProfile,
  fetchDeploymentReviewPreview,
  preloadAppliedProfile,
  preloadDeploymentReviewPreview,
  preloadGameProfiles,
  preloadStoredGames,
} from '@hooks';

import './GameProfileModsPanel.scss';

interface GameProfileModsPanelProps {
  appliedProfile: AppliedProfileSummary | null;
  applyProfilePath: string;
  gameID: number;
  gameMods: Mod[];
  isBusy: boolean;
  isGameModsLoading: boolean;
  isProfilesLoading: boolean;
  pendingProfileModToggleIDs: Record<string, boolean>;
  profile: ModProfile | null;
  profileMods: ProfileMod[];
  onAddModsToProfile: (profileID: number, modIDs: number[]) => Promise<void> | void;
  onRemoveModFromProfile: (profileID: number, modID: number) => void;
  onReorderProfileMods: (profileID: number, orderedModIDs: number[]) => void;
  onSetProfileModEnabled: (profileID: number, modID: number, enabled: boolean) => void;
}

export const GameProfileModsPanel = ({
  appliedProfile,
  applyProfilePath,
  gameID,
  gameMods,
  isBusy,
  isGameModsLoading,
  isProfilesLoading,
  pendingProfileModToggleIDs,
  profile,
  profileMods,
  onAddModsToProfile,
  onRemoveModFromProfile,
  onReorderProfileMods,
  onSetProfileModEnabled,
}: GameProfileModsPanelProps) => {
  const navigate = useNavigate();
  const { addErrorToast } = useToast();
  const preloadPromisesByProfileID = useRef<Record<number, Promise<void>>>({});
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isEnabledOnly, setIsEnabledOnly] = useState(false);
  const [pendingApplyProfileID, setPendingApplyProfileID] = useState<number | null>(null);
  const [selectedTagIDs, setSelectedTagIDs] = useState<number[]>([]);
  const assignedModIDs = useMemo(
    () => new Set(profileMods.map((profileMod) => profileMod.ModID)),
    [profileMods],
  );
  const modsByID = useMemo(() => new Map(gameMods.map((mod) => [mod.ID, mod])), [gameMods]);
  const assignedMods = useMemo(
    () =>
      profileMods.flatMap((profileMod) => {
        const mod = modsByID.get(profileMod.ModID);
        return mod === undefined ? [] : [mod];
      }),
    [modsByID, profileMods],
  );
  const tagsByModID = useMemo(
    () => Object.fromEntries(assignedMods.map((mod) => [mod.ID, mod.Tags])),
    [assignedMods],
  );
  const enabledModCount = useMemo(
    () => profileMods.filter((profileMod) => profileMod.Enabled).length,
    [profileMods],
  );
  const visibleProfileMods = useMemo(
    () =>
      profileMods.filter((profileMod) => {
        if (isEnabledOnly && !profileMod.Enabled) {
          return false;
        }

        const modTags = modsByID.get(profileMod.ModID)?.Tags ?? [];
        return selectedTagIDs.every((tagID) => modTags.some((tag) => tag.ID === tagID));
      }),
    [isEnabledOnly, modsByID, profileMods, selectedTagIDs],
  );
  const availableMods = useMemo(
    () => gameMods.filter((mod) => !assignedModIDs.has(mod.ID)),
    [assignedModIDs, gameMods],
  );
  const canOpenAddModal = !isBusy && !isGameModsLoading && availableMods.length > 0;
  const isSelectedProfileApplied = profile !== null && appliedProfile?.ProfileID === profile.ID;
  const isAnotherProfileApplied =
    profile !== null && appliedProfile !== null && !isSelectedProfileApplied;
  const blockedApplyTitle =
    appliedProfile === null
      ? undefined
      : `${appliedProfile.ProfileName} is applied. Restore vanilla before applying another profile.`;

  useEffect(() => {
    setIsAddModalOpen(false);
    setIsEnabledOnly(false);
    setSelectedTagIDs([]);
  }, [profile?.ID]);

  useEffect(() => {
    if (profile === null) {
      return;
    }

    delete preloadPromisesByProfileID.current[profile.ID];
  }, [profile?.ID, profileMods]);

  useEffect(() => {
    const availableTagIDs = new Set(assignedMods.flatMap((mod) => mod.Tags.map((tag) => tag.ID)));
    setSelectedTagIDs((currentTagIDs) =>
      currentTagIDs.filter((tagID) => availableTagIDs.has(tagID)),
    );
  }, [assignedMods]);

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

  const warmApplyDestination = (profileID: number) => {
    preloadPromisesByProfileID.current[profileID] ??= Promise.all([
      preloadStoredGames(),
      preloadGameProfiles(gameID),
      preloadAppliedProfile(gameID),
      preloadDeploymentReviewPreview(profileID),
    ]).then(() => undefined);

    preloadPromisesByProfileID.current[profileID].catch(() => undefined);
  };

  const navigateAfterApplyPreload = async (
    event: MouseEvent<HTMLAnchorElement>,
    profileID: number,
  ) => {
    event.preventDefault();

    if (isBusy || pendingApplyProfileID !== null) {
      return;
    }

    setPendingApplyProfileID(profileID);

    try {
      await Promise.all([
        preloadStoredGames(),
        preloadGameProfiles(gameID),
        fetchAppliedProfile(gameID),
        fetchDeploymentReviewPreview(profileID),
      ]);
      navigate(`${applyProfilePath}/${profileID}`);
    } catch (error) {
      delete preloadPromisesByProfileID.current[profileID];
      addErrorToast(error);
    } finally {
      setPendingApplyProfileID(null);
    }
  };

  if (profile === null) {
    return (
      <section className="game-profile-mods-panel" aria-label="Profile mods">
        <StateBlock
          className="game-profile-mods-panel-empty"
          message={
            isProfilesLoading ? 'Loading profile details...' : 'Create a profile to configure mods.'
          }
        />
      </section>
    );
  }

  return (
    <section className="game-profile-mods-panel" aria-label={`${profile.Name} mods`}>
      <div className="game-profile-mods-panel-body">
        {isGameModsLoading && gameMods.length === 0 && (
          <StateBlock
            className="game-profile-mods-panel-empty"
            message="Loading imported mods..."
          />
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
          <>
            {visibleProfileMods.length === 0 ? (
              <StateBlock
                className="game-profile-mods-panel-empty game-profile-mods-panel-empty-row"
                message="No assigned mods match the active filters."
              />
            ) : (
              <GameProfileAssignedModsList
                canReorder={!isEnabledOnly && selectedTagIDs.length === 0}
                isBusy={isBusy}
                mods={visibleProfileMods}
                pendingToggleIDs={pendingProfileModToggleIDs}
                tagsByModID={tagsByModID}
                onMoveMod={handleMoveProfileMod}
                onReorderMods={(orderedModIDs) => onReorderProfileMods(profile.ID, orderedModIDs)}
                onRemoveMod={(modID) => onRemoveModFromProfile(profile.ID, modID)}
                onSetModEnabled={(modID, enabled) =>
                  onSetProfileModEnabled(profile.ID, modID, enabled)
                }
              />
            )}
          </>
        )}
      </div>

      <div className="game-profile-mods-panel-footer">
        <div className="game-profile-mods-panel-footer-actions">
          <button
            className="game-profile-mods-panel-add-button"
            disabled={!canOpenAddModal}
            onClick={() => setIsAddModalOpen(true)}
            type="button"
          >
            <Plus className="game-profile-mods-panel-icon" aria-hidden="true" />
            <span>Add Mods from Library</span>
          </button>

          {profileMods.length > 0 && (
            <>
              <GameProfileModsFilter
                enabledCount={enabledModCount}
                isEnabledOnly={isEnabledOnly}
                totalCount={profileMods.length}
                onEnabledOnlyChange={setIsEnabledOnly}
              />
              <ModTagFilter
                candidateMods={assignedMods}
                popoverPlacement="above"
                selectedTagIDs={selectedTagIDs}
                variant="profile-footer"
                onChange={setSelectedTagIDs}
              />
            </>
          )}
        </div>

        <div className="game-profile-mods-panel-footer-primary">
          {isSelectedProfileApplied ? (
            <Link
              className={
                isBusy || pendingApplyProfileID === profile.ID
                  ? 'game-profile-mods-panel-review-button button-main game-profile-mods-panel-link-disabled'
                  : 'game-profile-mods-panel-review-button button-main'
              }
              to={`${applyProfilePath}/${profile.ID}`}
              onClick={(event) => void navigateAfterApplyPreload(event, profile.ID)}
              onFocus={() => warmApplyDestination(profile.ID)}
              onMouseDown={() => warmApplyDestination(profile.ID)}
              onPointerEnter={() => warmApplyDestination(profile.ID)}
              aria-disabled={isBusy || pendingApplyProfileID === profile.ID}
            >
              {pendingApplyProfileID === profile.ID ? (
                <InlineLoading label="Review deployment" />
              ) : (
                <>
                  <FileSearch className="game-profile-mods-panel-icon" aria-hidden="true" />
                  <span>Review deployment</span>
                </>
              )}
            </Link>
          ) : isAnotherProfileApplied ? (
            <button
              className="game-profile-mods-panel-apply-button button-main"
              disabled
              title={blockedApplyTitle}
              type="button"
            >
              <CheckCircle2 className="game-profile-mods-panel-icon" aria-hidden="true" />
              <span>Another Profile Applied</span>
            </button>
          ) : (
            <Link
              className={
                isBusy || pendingApplyProfileID === profile.ID
                  ? 'game-profile-mods-panel-apply-button button-main game-profile-mods-panel-link-disabled'
                  : 'game-profile-mods-panel-apply-button button-main'
              }
              to={`${applyProfilePath}/${profile.ID}`}
              onClick={(event) => void navigateAfterApplyPreload(event, profile.ID)}
              onFocus={() => warmApplyDestination(profile.ID)}
              onMouseDown={() => warmApplyDestination(profile.ID)}
              onPointerEnter={() => warmApplyDestination(profile.ID)}
              aria-disabled={isBusy || pendingApplyProfileID === profile.ID}
            >
              {pendingApplyProfileID === profile.ID ? (
                <InlineLoading label="Apply Profile" />
              ) : (
                <>
                  <CheckCircle2 className="game-profile-mods-panel-icon" aria-hidden="true" />
                  <span>Apply Profile</span>
                </>
              )}
            </Link>
          )}
        </div>
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
