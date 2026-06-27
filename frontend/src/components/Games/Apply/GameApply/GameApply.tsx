import { useCallback, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ArrowLeft, CheckCircle2 } from 'lucide-react';

import {
  ApplyProfileOperationPlan,
  BuildProfileOperationPlan,
} from '@bindings/github.com/phergul/fiach/internal/services/profileservice';
import type { ApplyOperationPlanResult, OperationPlan } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { useToast } from '@components/Common/Toast/Toast';
import { DeploymentReview } from '@components/Deployment/DeploymentReview/DeploymentReview';
import { DeploymentSummaryBar } from '@components/Deployment/DeploymentSummary/DeploymentSummary';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import {
  useAppliedProfile,
  useDeploymentReviewPreview,
  useGameArtwork,
  useGameProfiles,
  useStoredGames,
} from '@hooks';

import './GameApply.scss';
import { getApplyDisabledTitle, getDeploymentReviewDescription } from './gameApplyCopy';

const parseGameID = (gameID: string | undefined) => {
  if (gameID === undefined || gameID.trim() === '') {
    return null;
  }

  const parsedGameID = Number(gameID);
  if (!Number.isInteger(parsedGameID) || parsedGameID <= 0) {
    return null;
  }

  return parsedGameID;
};

const parseProfileID = (profileID: string | undefined) => {
  if (profileID === undefined || profileID.trim() === '') {
    return null;
  }

  const parsedProfileID = Number(profileID);
  if (!Number.isInteger(parsedProfileID) || parsedProfileID <= 0) {
    return null;
  }

  return parsedProfileID;
};

const buildApplySuccessMessage = (result: ApplyOperationPlanResult) => {
  if (result.CompletedCount === 0) {
    return 'No operations were needed.';
  }
  if (result.CompletedCount === 1) {
    return 'Applied 1 operation.';
  }

  return `Applied ${result.CompletedCount} operations.`;
};

const buildApplyFailureMessage = (result: ApplyOperationPlanResult) => {
  const failedResult = result.Results.find((operationResult) => operationResult.Error !== null);
  const failure = failedResult?.Error ?? 'Apply stopped before all operations completed.';

  return `Apply stopped: ${failure} Completed ${result.CompletedCount}, skipped ${result.SkippedCount}.`;
};

export const GameApply = () => {
  const { gameId, profileId } = useParams();
  const navigate = useNavigate();
  const { addErrorToast, addToast } = useToast();
  const [isApplyConfirmOpen, setIsApplyConfirmOpen] = useState(false);
  const [isApplyPending, setIsApplyPending] = useState(false);
  const [isPlanLoading, setIsPlanLoading] = useState(false);
  const { games, isLoading, isScanning, loadError, retryLoadGames } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const parsedProfileID = parseProfileID(profileId);
  const game =
    parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const appliedProfileManager = useAppliedProfile(game?.ID ?? null);
  const selectedProfile =
    parsedProfileID === null
      ? null
      : (profileManager.profiles.find((profile) => profile.ID === parsedProfileID) ?? null);
  const {
    isLoading: isPreviewLoading,
    loadError: previewLoadError,
    previewHash,
    refreshPreview,
    rootChildren,
    summary,
  } = useDeploymentReviewPreview(selectedProfile?.ID ?? null);
  const { artworkSource: heroArtworkSource, handleArtworkError: handleHeroArtworkError } =
    useGameArtwork(game?.Source === 'steam' && game.SourceID ? game.SourceID : '', 'hero');
  const isWaitingForGame = (isLoading || isScanning) && game === undefined;
  const hasLoadError = loadError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;
  const gameDetailsPath = parsedGameID === null ? '/library' : `/library/${parsedGameID}`;
  const appliedProfileName = appliedProfileManager.appliedProfile?.ProfileName ?? null;
  const isSameProfileApplied =
    selectedProfile !== null &&
    appliedProfileManager.appliedProfile?.ProfileID === selectedProfile.ID;
  const isAnotherProfileApplied =
    appliedProfileName !== null && !isSameProfileApplied;
  const previewAvailable = summary !== null && previewHash !== '';
  const canStartApply =
    selectedProfile !== null &&
    appliedProfileName === null &&
    !appliedProfileManager.isLoading &&
    appliedProfileManager.loadError === null &&
    previewAvailable &&
    summary.CanApply &&
    !isPreviewLoading &&
    !isApplyPending &&
    !isPlanLoading;
  const applyTitle =
    selectedProfile === null
      ? 'Select a profile before applying.'
      : getApplyDisabledTitle(
          isSameProfileApplied,
          isAnotherProfileApplied,
          appliedProfileName,
          summary?.CanApply ?? false,
          appliedProfileManager.isLoading,
          appliedProfileManager.loadError,
          isPreviewLoading,
          isApplyPending,
          isPlanLoading,
          previewAvailable,
        );
  const confirmMessage =
    selectedProfile === null
      ? 'Review the deployment preview before applying this profile.'
      : `Apply ${selectedProfile.Name}? This will change the installed game files directly. Replaced files will be backed up for restore.`;

  const handlePreviewRefreshNeeded = useCallback(() => {
    refreshPreview().catch(() => undefined);
  }, [refreshPreview]);

  const openApplyConfirm = () => {
    if (canStartApply) {
      setIsApplyConfirmOpen(true);
    }
  };

  const closeApplyConfirm = () => {
    if (!isApplyPending && !isPlanLoading) {
      setIsApplyConfirmOpen(false);
    }
  };

  const confirmApply = async () => {
    if (selectedProfile === null || isApplyPending || isPlanLoading) {
      return;
    }

    setIsPlanLoading(true);

    let plan: OperationPlan;
    try {
      plan = await BuildProfileOperationPlan(selectedProfile.ID);
    } catch (error) {
      setIsPlanLoading(false);
      addErrorToast(error);
      return;
    }

    if (!plan.CanApply) {
      setIsPlanLoading(false);
      addErrorToast(new Error('The apply plan has blocking issues. Refresh the preview and try again.'));
      return;
    }

    setIsApplyPending(true);

    try {
      const result = await ApplyProfileOperationPlan(selectedProfile.ID, plan);
      if (result.Success) {
        await appliedProfileManager.refreshAppliedProfile();
      }
      setIsApplyConfirmOpen(false);
      addToast({
        message: result.Success
          ? buildApplySuccessMessage(result)
          : buildApplyFailureMessage(result),
        tone: result.Success ? 'success' : 'error',
      });
      if (result.Success) {
        navigate(gameDetailsPath);
      }
    } catch (error) {
      setIsApplyConfirmOpen(false);
      addErrorToast(error);
    } finally {
      setIsApplyPending(false);
      setIsPlanLoading(false);
    }
  };

  const showDeploymentReview =
    selectedProfile !== null &&
    !isPreviewLoading &&
    previewLoadError === null &&
    previewHash !== '';

  const hasBackdrop = heroArtworkSource !== '' && !showDeploymentReview;

  return (
    <section
      className={
        showDeploymentReview
          ? 'game-apply game-apply-deployment'
          : hasBackdrop
            ? 'game-apply game-apply-with-backdrop'
            : 'game-apply'
      }
      aria-label="Apply profile"
    >
      <div className="game-apply-toolbar">
        <div className="game-apply-toolbar-start">
          <Link className="game-apply-back-link" to={gameDetailsPath}>
            <ArrowLeft className="game-apply-toolbar-icon" aria-hidden="true" />
            Back
          </Link>

          {game !== undefined && selectedProfile !== null && !showDeploymentReview && (
            <div className="game-apply-context">
              <p className="game-apply-context-title">
                {game.Name} · Apply {selectedProfile.Name}
              </p>
              <p className="game-apply-context-description">
                {getDeploymentReviewDescription(
                  isSameProfileApplied,
                  isAnotherProfileApplied,
                  appliedProfileName,
                )}
              </p>
            </div>
          )}
        </div>

        <button
          className="game-apply-toolbar-button"
          disabled={!canStartApply}
          onClick={openApplyConfirm}
          title={applyTitle}
          type="button"
        >
          <CheckCircle2 className="game-apply-toolbar-icon" aria-hidden="true" />
          <span>{isApplyPending || isPlanLoading ? 'Applying' : 'Apply'}</span>
        </button>
      </div>

      <ConfirmDialog
        cancelLabel="Cancel"
        confirmLabel="Confirm apply"
        confirmTone="default"
        isBusy={isApplyPending || isPlanLoading}
        isOpen={isApplyConfirmOpen}
        message={confirmMessage}
        onCancel={closeApplyConfirm}
        onConfirm={confirmApply}
        title="Confirm apply"
      />

      {hasBackdrop && (
        <div className="game-apply-backdrop" aria-hidden="true">
          <img
            className="game-apply-backdrop-image"
            src={heroArtworkSource}
            alt=""
            onError={handleHeroArtworkError}
          />
        </div>
      )}

      {isWaitingForGame && <GameDetailsState />}

      {hasLoadError && (
        <GameDetailsState
          actionLabel="Retry"
          linkLabel="Return to library"
          message={loadError}
          onAction={retryLoadGames}
          title="Could not load game."
        />
      )}

      {hasNotFound && (
        <GameDetailsState
          message="This game is not currently available in the library."
          title="Game not found."
        />
      )}

      {game !== undefined && (
        <>
          {!showDeploymentReview && (
            <div className="game-apply-heading">
              <h2 className="game-apply-title">
                {selectedProfile === null ? 'Apply profile' : `Apply ${selectedProfile.Name}`}
              </h2>
              <p className="game-apply-description">
                {getDeploymentReviewDescription(
                  isSameProfileApplied,
                  isAnotherProfileApplied,
                  appliedProfileName,
                )}
              </p>
            </div>
          )}

          {showDeploymentReview && (
            <header className="game-apply-page-header">
              <div className="game-apply-breadcrumbs" aria-hidden="true" />
              <div className="game-apply-heading">
                <h2 className="game-apply-title">Deployment preview</h2>
                <p className="game-apply-description">
                  {getDeploymentReviewDescription(
                    isSameProfileApplied,
                    isAnotherProfileApplied,
                    appliedProfileName,
                  )}
                </p>
              </div>
            </header>
          )}

          {showDeploymentReview && summary !== null && <DeploymentSummaryBar summary={summary} />}

          {profileManager.isLoading && <GameDetailsState title="Loading selected profile." />}

          {!profileManager.isLoading && profileManager.loadError !== null && (
            <GameDetailsState
              actionLabel="Retry"
              message={profileManager.loadError}
              onAction={profileManager.refreshProfiles}
              title="Could not load profiles."
            />
          )}

          {!profileManager.isLoading &&
            profileManager.loadError === null &&
            parsedProfileID === null && (
              <GameDetailsState
                message="Choose a profile from the game details screen before opening the deployment review."
                title="No selected profile."
              />
            )}

          {!profileManager.isLoading &&
            profileManager.loadError === null &&
            parsedProfileID !== null &&
            selectedProfile === null && (
              <GameDetailsState
                message="This profile is not currently available for the selected game."
                title="Profile not found."
              />
            )}

          {selectedProfile !== null && isPreviewLoading && (
            <GameDetailsState title="Building deployment preview." />
          )}

          {selectedProfile !== null && !isPreviewLoading && previewLoadError !== null && (
            <GameDetailsState
              actionLabel="Retry"
              message={previewLoadError}
              onAction={() => {
                refreshPreview().catch(() => undefined);
              }}
              title="Could not build deployment preview."
            />
          )}

          {showDeploymentReview && (
              <DeploymentReview
                gameInstallPath={game.InstallPath}
                gameName={game.Name}
                onPreviewRefreshNeeded={handlePreviewRefreshNeeded}
                planMode={summary?.PlanMode ?? 'first_apply'}
                previewHash={previewHash}
                rootChildren={rootChildren}
              />
            )}
        </>
      )}
    </section>
  );
};
