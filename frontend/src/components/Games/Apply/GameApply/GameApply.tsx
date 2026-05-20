import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, CheckCircle2 } from 'lucide-react';

import {
  OperationType,
  PlanIssueSeverity,
  type OperationPlan,
} from '@bindings/github.com/phergul/mod-manager/internal/operationplan/models';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { GameApplyReview } from '@components/Games/Apply/GameApplyReview/GameApplyReview';
import { GameApplySummary, type GameApplySummaryItem } from '@components/Games/Apply/GameApplySummary/GameApplySummary';
import { useGameArtwork, useGameProfiles, useProfileOperationPlan, useStoredGames } from '@hooks';

import './GameApply.scss';

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

const getApplyDisabledTitle = (plan: OperationPlan | null, isPlanLoading: boolean) => {
  if (isPlanLoading) {
    return 'Operation plan is loading.';
  }
  if (plan !== null && !plan.CanApply) {
    return 'Resolve blocking issues before applying this profile.';
  }

  return 'Profile apply execution is not connected yet.';
};

export const GameApply = () => {
  const { gameId } = useParams();
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
  const applyTitle = activeProfile === null
    ? 'Select an active profile before applying.'
    : getApplyDisabledTitle(plan, isPlanLoading);

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

        <button className="game-apply-toolbar-button" disabled title={applyTitle} type="button">
          <CheckCircle2 className="game-apply-toolbar-icon" aria-hidden="true" />
          <span>Apply</span>
        </button>
      </div>

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
              Review planned file and folder changes before applying the active profile.
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
