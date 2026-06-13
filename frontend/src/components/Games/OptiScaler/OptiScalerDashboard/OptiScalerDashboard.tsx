import { useState } from 'react';

import { ArrowLeft, RefreshCw } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';

import { useToast } from '@components/Common/Toast/Toast';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { OptiScalerRecoveryPanel } from '@components/Games/OptiScaler/OptiScalerRecoveryPanel/OptiScalerRecoveryPanel';
import {
  OptiScalerTargetList,
  type OptiScalerSelection,
} from '@components/Games/OptiScaler/OptiScalerTargetList/OptiScalerTargetList';
import { OptiScalerWizard } from '@components/Games/OptiScaler/OptiScalerWizard/OptiScalerWizard';
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
  const [selection, setSelection] = useState<OptiScalerSelection | null>(null);
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
    <section className="optiscaler-dashboard" aria-label="OptiScaler management">
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
              <h1>OptiScaler</h1>
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

          {optiScaler.isLoading && optiScaler.targets.length === 0 && optiScaler.candidates.length === 0 && (
            <GameDetailsState title="Discovering OptiScaler targets." />
          )}

          {!optiScaler.isLoading && optiScaler.loadError !== null && (
            <GameDetailsState
              actionLabel="Retry"
              message={optiScaler.loadError}
              onAction={() => void optiScaler.refresh()}
              title="Could not load OptiScaler state."
            />
          )}

          {optiScaler.loadError === null && (
            <OptiScalerTargetList
              candidates={optiScaler.candidates}
              disabled={isRecoveryRequired}
              onSelect={setSelection}
              release={optiScaler.release}
              targets={optiScaler.targets}
            />
          )}

          {selection !== null && !isRecoveryRequired && (
            <OptiScalerWizard
              gameID={game.ID}
              onClose={() => setSelection(null)}
              onRecoveryRequired={optiScaler.refresh}
              onRefresh={optiScaler.refresh}
              selection={selection}
            />
          )}
        </>
      )}
    </section>
  );
};
