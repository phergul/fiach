import { useState } from 'react';

import { ArrowLeft, RefreshCw } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';

import { Action } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import { useToast } from '@components/Common/Toast/Toast';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { OptiScalerDetail } from '@components/Games/OptiScaler/OptiScalerDetail/OptiScalerDetail';
import { OptiScalerRecoveryPanel } from '@components/Games/OptiScaler/OptiScalerRecoveryPanel/OptiScalerRecoveryPanel';
import {
  optiScalerSelectionKey,
  OptiScalerTargetList,
  type OptiScalerSelection,
} from '@components/Games/OptiScaler/OptiScalerTargetList/OptiScalerTargetList';
import {
  OptiScalerWizard,
  type OptiScalerOperationSelection,
} from '@components/Games/OptiScaler/OptiScalerWizard/OptiScalerWizard';
import { useGameArtwork, useGameOptiScaler, useStoredGames } from '@hooks';

import './OptiScalerDashboard.scss';

const parseGameID = (gameID: string | undefined) => {
  const parsed = Number(gameID);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null;
};

export const OptiScalerDashboard = () => {
  const { gameId } = useParams();
  const { addErrorToast, addToast } = useToast();
  const { games, isLoading: isLoadingGames, isScanning, loadError: gamesError, retryLoadGames } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const optiScaler = useGameOptiScaler(game?.ID ?? null);
  const [targetSelection, setTargetSelection] = useState<OptiScalerSelection | null>(null);
  const [operationSelection, setOperationSelection] = useState<OptiScalerOperationSelection | null>(null);
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
  const isWaitingForGame = (isLoadingGames || isScanning) && game === undefined;
  const hasLoadError = gamesError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;
  const gameDetailsPath = parsedGameID === null ? '/library' : `/library/${parsedGameID}`;
  const isRecoveryRequired = optiScaler.recovery?.required === true;
  const detectedCandidateCount = optiScaler.candidates.filter((candidate) => !candidate.managed).length;

  const selectTarget = (selection: OptiScalerSelection) => {
    setTargetSelection(selection);
    setOperationSelection(null);
  };

  const startAction = (action: Action) => {
    if (targetSelection !== null) {
      setOperationSelection({ ...targetSelection, action });
    }
  };

  const rollbackRecovery = async () => {
    try {
      const result = await optiScaler.rollbackRecovery();
      if (result !== null) {
        addToast({ message: result.message, tone: result.success ? 'success' : 'error' });
      }
    } catch (error) {
      addErrorToast(error);
    }
  };

  return (
    <section
      className={heroArtworkSource === ''
        ? 'optiscaler-dashboard'
        : 'optiscaler-dashboard optiscaler-dashboard-with-backdrop'}
      aria-label="OptiScaler management"
    >
      <div className="optiscaler-dashboard-toolbar">
        <Link className="optiscaler-dashboard-back-link" to={gameDetailsPath}>
          <ArrowLeft aria-hidden="true" />
          Back
        </Link>
        <button
          disabled={optiScaler.isLoading}
          onClick={() => void optiScaler.refresh()}
          type="button"
        >
          <RefreshCw aria-hidden="true" />
          Refresh
        </button>
      </div>

      {heroArtworkSource !== '' && (
        <div className="optiscaler-dashboard-backdrop" aria-hidden="true">
          <img
            className="optiscaler-dashboard-backdrop-image"
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
          message={gamesError}
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

          <div className="optiscaler-dashboard-heading">
            <div>
              <h2>OptiScaler</h2>
              <p>Manage each executable directory independently and review every file change before apply.</p>
            </div>
            <div className="optiscaler-dashboard-release">
              <span>Stable release</span>
              <strong>
                {optiScaler.isReleaseLoading
                  ? 'Checking'
                  : optiScaler.release?.version || optiScaler.release?.tag || 'Unavailable'}
              </strong>
            </div>
          </div>

          {optiScaler.releaseError !== null && (
            <p className="optiscaler-dashboard-error">{optiScaler.releaseError}</p>
          )}

          {isRecoveryRequired && optiScaler.recovery !== null && (
            <OptiScalerRecoveryPanel
              isRollingBack={optiScaler.isRollingBack}
              onRollback={() => void rollbackRecovery()}
              recovery={optiScaler.recovery}
            />
          )}

          <div className="optiscaler-dashboard-workspace">
            <aside className="optiscaler-dashboard-sidebar" aria-label="OptiScaler targets">
              {optiScaler.isLoading && optiScaler.targets.length === 0 && optiScaler.candidates.length === 0 ? (
                <GameDetailsState title="Discovering OptiScaler targets." />
              ) : optiScaler.loadError !== null ? (
                <GameDetailsState
                  actionLabel="Retry"
                  message={optiScaler.loadError}
                  onAction={() => void optiScaler.refresh()}
                  title="Could not load OptiScaler state."
                />
              ) : (
                <OptiScalerTargetList
                  candidates={optiScaler.candidates}
                  disabled={isRecoveryRequired}
                  onSelect={selectTarget}
                  release={optiScaler.release}
                  selectedKey={targetSelection === null ? null : optiScalerSelectionKey(targetSelection)}
                  targets={optiScaler.targets}
                />
              )}
            </aside>
            <main className="optiscaler-dashboard-detail">
              {operationSelection !== null && !isRecoveryRequired ? (
                <OptiScalerWizard
                  gameID={game.ID}
                  onClose={() => setOperationSelection(null)}
                  onRecoveryRequired={optiScaler.refresh}
                  onRefresh={optiScaler.refresh}
                  selection={operationSelection}
                />
              ) : (
                <OptiScalerDetail
                  candidateCount={detectedCandidateCount}
                  managedCount={optiScaler.targets.length}
                  onStartAction={startAction}
                  release={optiScaler.release}
                  selection={targetSelection}
                />
              )}
            </main>
          </div>
        </>
      )}
    </section>
  );
};
