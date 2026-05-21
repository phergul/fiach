import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, CheckCircle2 } from 'lucide-react';

import { ConfirmProfileOperationPlan } from '@bindings/github.com/phergul/mod-manager/internal/services/profileservice';
import {
  OperationType,
  PlanIssueSeverity,
  type OperationPlan,
} from '@bindings/github.com/phergul/mod-manager/internal/operationplan/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { useToast } from '@components/Common/Toast/Toast';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { GameApplyReview } from '@components/Games/Apply/GameApplyReview/GameApplyReview';
import { GameApplySummary, type GameApplySummaryItem } from '@components/Games/Apply/GameApplySummary/GameApplySummary';
import { useGameArtwork, useGameProfiles, useProfileOperationPlan, useStoredGames } from '@hooks';
import { getErrorMessage } from '@utils';

import './GameApply.scss';

const applyExecutionHandoffMessage = 'apply execution is reserved for Epic 8';

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

const countOperations = (plan: OperationPlan, type: OperationType) => {
  return plan.Operations.filter((operation) => operation.Type === type).length;
};

const countIssues = (plan: OperationPlan, severity: PlanIssueSeverity) => {
  return plan.Issues.filter((issue) => issue.Severity === severity).length;
};

const buildSummaryItems = (plan: OperationPlan | null): GameApplySummaryItem[] => {
  return [
    {
      label: 'Files to add',
      value: plan === null ? 0 : countOperations(plan, OperationType.OperationTypeCopy),
    },
    {
      label: 'Files to replace',
      value: plan === null ? 0 : countOperations(plan, OperationType.OperationTypeReplace),
    },
    {
      label: 'Folders to create',
      value: plan === null ? 0 : countOperations(plan, OperationType.OperationTypeCreateDirectory),
    },
    {
      label: 'Blocking issues',
      value: plan === null ? 0 : countIssues(plan, PlanIssueSeverity.PlanIssueSeverityError),
    },
    {
      label: 'Warnings',
      value: plan === null ? 0 : countIssues(plan, PlanIssueSeverity.PlanIssueSeverityWarning),
    },
  ];
};

const getApplyDisabledTitle = (
  plan: OperationPlan | null,
  isPlanLoading: boolean,
  isApplyHandoffPending: boolean,
) => {
  if (isApplyHandoffPending) {
    return 'Apply confirmation is already in progress.';
  }
  if (isPlanLoading) {
    return 'Operation plan is loading.';
  }
  if (plan !== null && !plan.CanApply) {
    return 'Resolve blocking issues before applying this profile.';
  }
  if (plan === null) {
    return 'Operation plan is not ready yet.';
  }

  return 'Confirm before applying this profile.';
};

export const GameApply = () => {
  const { gameId } = useParams();
  const { addToast } = useToast();
  const [isApplyConfirmOpen, setIsApplyConfirmOpen] = useState(false);
  const [isApplyHandoffPending, setIsApplyHandoffPending] = useState(false);
  const { games, isLoading, isScanning, loadError, retryLoadGames } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const activeProfile = profileManager.activeProfile;
  const {
    isLoading: isPlanLoading,
    loadError: planLoadError,
    plan,
    refreshPlan,
  } = useProfileOperationPlan(activeProfile?.ID ?? null);
  const {
    artworkSource: heroArtworkSource,
    handleArtworkError: handleHeroArtworkError,
  } = useGameArtwork(
    game?.Source === 'steam' && game.SourceID ? game.SourceID : '',
    'hero',
  );
  const {
    artworkSource: logoArtworkSource,
    handleArtworkError: handleLogoArtworkError,
  } = useGameArtwork(
    game?.Source === 'steam' && game.SourceID ? game.SourceID : '',
    'logo',
  );
  const isWaitingForGame = (isLoading || isScanning) && game === undefined;
  const hasLoadError = loadError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;
  const gameDetailsPath = parsedGameID === null ? '/library' : `/library/${parsedGameID}`;
  const summaryItems = buildSummaryItems(plan);
  const canStartApplyHandoff = activeProfile !== null &&
    plan !== null &&
    plan.CanApply &&
    !isPlanLoading &&
    !isApplyHandoffPending;
  const applyTitle = activeProfile === null
    ? 'Select an active profile before applying.'
    : getApplyDisabledTitle(plan, isPlanLoading, isApplyHandoffPending);
  const confirmMessage = activeProfile === null || plan === null
    ? 'Review the operation plan before applying this profile.'
    : `This will alter game files for the installed game. Are you sure you want to apply ${activeProfile.Name}?`;

  const openApplyConfirm = () => {
    if (canStartApplyHandoff) {
      setIsApplyConfirmOpen(true);
    }
  };

  const closeApplyConfirm = () => {
    if (!isApplyHandoffPending) {
      setIsApplyConfirmOpen(false);
    }
  };

  const confirmApplyHandoff = async () => {
    if (activeProfile === null || plan === null || isApplyHandoffPending) {
      return;
    }

    setIsApplyHandoffPending(true);

    try {
      await ConfirmProfileOperationPlan(activeProfile.ID, plan);
      setIsApplyConfirmOpen(false);
      addToast({
        message: 'Operation plan confirmed.',
        tone: 'success',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      setIsApplyConfirmOpen(false);
      addToast({
        message,
        tone: message.includes(applyExecutionHandoffMessage) ? 'info' : 'error',
      });
    } finally {
      setIsApplyHandoffPending(false);
    }
  };

  return (
    <section
      className={heroArtworkSource === '' ? 'game-apply' : 'game-apply game-apply-with-backdrop'}
      aria-label="Apply profile"
    >
      <div className="game-apply-toolbar">
        <Link className="game-apply-back-link" to={gameDetailsPath}>
          <ArrowLeft className="game-apply-toolbar-icon" aria-hidden="true" />
          Back
        </Link>

        <button
          className="game-apply-toolbar-button"
          disabled={!canStartApplyHandoff}
          onClick={openApplyConfirm}
          title={applyTitle}
          type="button"
        >
          <CheckCircle2 className="game-apply-toolbar-icon" aria-hidden="true" />
          <span>{isApplyHandoffPending ? 'Confirming' : 'Apply'}</span>
        </button>
      </div>

      <ConfirmDialog
        cancelLabel="Cancel"
        confirmLabel="Confirm apply"
        confirmTone="default"
        isBusy={isApplyHandoffPending}
        isOpen={isApplyConfirmOpen}
        message={confirmMessage}
        onCancel={closeApplyConfirm}
        onConfirm={confirmApplyHandoff}
        title="Confirm apply"
      />

      {heroArtworkSource !== '' && (
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
          <GameDetailsHeader
            game={game}
            logoArtworkSource={logoArtworkSource}
            onLogoArtworkError={handleLogoArtworkError}
          />

          <div className="game-apply-heading">
            <h2 className="game-apply-title">
              {activeProfile === null ? 'Apply profile' : `Apply ${activeProfile.Name}`}
            </h2>
            <p className="game-apply-description">
              Review planned file and folder changes before applying.
            </p>
          </div>

          <GameApplySummary items={summaryItems} />

          {profileManager.isLoading && <GameDetailsState title="Loading active profile." />}

          {!profileManager.isLoading && profileManager.loadError !== null && (
            <GameDetailsState
              actionLabel="Retry"
              message={profileManager.loadError}
              onAction={profileManager.refreshProfiles}
              title="Could not load profiles."
            />
          )}

          {!profileManager.isLoading && profileManager.loadError === null && activeProfile === null && (
            <GameDetailsState
              message="Choose an active profile from the game details screen before opening the apply preview."
              title="No active profile."
            />
          )}

          {activeProfile !== null && isPlanLoading && <GameDetailsState title="Building operation plan." />}

          {activeProfile !== null && !isPlanLoading && planLoadError !== null && (
            <GameDetailsState
              actionLabel="Retry"
              message={planLoadError}
              onAction={() => {
                refreshPlan().catch(() => undefined);
              }}
              title="Could not build operation plan."
            />
          )}

          {activeProfile !== null && !isPlanLoading && planLoadError === null && plan !== null && (
            <GameApplyReview gameInstallPath={game.InstallPath} gameName={game.Name} plan={plan} />
          )}
        </>
      )}
    </section>
  );
};
