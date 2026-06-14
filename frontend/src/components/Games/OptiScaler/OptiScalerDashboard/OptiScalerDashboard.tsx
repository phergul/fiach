import { useEffect, useState } from 'react';

import { ArrowLeft } from 'lucide-react';
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom';

import { useToast } from '@components/Common/Toast/Toast';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { OptiScalerExecutableTable } from '@components/Games/OptiScaler/OptiScalerExecutableTable/OptiScalerExecutableTable';
import { OptiScalerPageHeader } from '@components/Games/OptiScaler/OptiScalerPageHeader/OptiScalerPageHeader';
import { OptiScalerRecoveryPanel } from '@components/Games/OptiScaler/OptiScalerRecoveryPanel/OptiScalerRecoveryPanel';
import {
  OptiScalerReShadeSession,
  type OptiScalerReShadeRequest,
} from '@components/Games/OptiScaler/OptiScalerReShadeSession/OptiScalerReShadeSession';
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
  const location = useLocation();
  const navigate = useNavigate();
  const { addErrorToast, addToast } = useToast();
  const { games, isLoading: isLoadingGames, isScanning, loadError: gamesError, retryLoadGames } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const optiScaler = useGameOptiScaler(game?.ID ?? null);
  const [operationSelection, setOperationSelection] = useState<OptiScalerOperationSelection | null>(null);
  const initialReShadeRequest = (
    location.state as { reShadeCoordination?: OptiScalerReShadeRequest } | null
  )?.reShadeCoordination ?? null;
  const [reShadeRequest, setReShadeRequest] = useState<OptiScalerReShadeRequest | null>(
    initialReShadeRequest,
  );
  const [isReShadeSessionActive, setIsReShadeSessionActive] = useState(
    initialReShadeRequest !== null,
  );
  useEffect(() => {
    if (initialReShadeRequest !== null) {
      navigate(location.pathname, { replace: true, state: null });
    }
  }, [initialReShadeRequest, location.pathname, navigate]);
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

          <OptiScalerPageHeader
            isLoading={optiScaler.isLoading}
            onRefresh={() => void optiScaler.refresh()}
            release={optiScaler.release}
          />

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

          <main className="optiscaler-dashboard-content">
            <OptiScalerReShadeSession
              gameID={game.ID}
              onActiveChange={setIsReShadeSessionActive}
              onRefresh={optiScaler.refresh}
              request={reShadeRequest}
              targets={optiScaler.targets}
            />
            {operationSelection !== null && !isRecoveryRequired && !isReShadeSessionActive ? (
              <OptiScalerWizard
                gameID={game.ID}
                onClose={() => setOperationSelection(null)}
                onRecoveryRequired={optiScaler.refresh}
                onRefresh={optiScaler.refresh}
                selection={operationSelection}
              />
            ) : optiScaler.isLoading && optiScaler.targets.length === 0 && optiScaler.candidates.length === 0 ? (
              <GameDetailsState title="Discovering OptiScaler targets." />
            ) : optiScaler.loadError !== null ? (
              <GameDetailsState
                actionLabel="Retry"
                message={optiScaler.loadError}
                onAction={() => void optiScaler.refresh()}
                title="Could not load OptiScaler state."
              />
            ) : (
              <OptiScalerExecutableTable
                candidates={optiScaler.candidates}
                disabled={isRecoveryRequired || isReShadeSessionActive}
                onStartOperation={setOperationSelection}
                onStartReShade={(target, variant) => {
                  setOperationSelection(null);
                  setReShadeRequest({
                    targetRelativePath: target.TargetRelativePath,
                    variant,
                  });
                }}
                release={optiScaler.release}
                targets={optiScaler.targets}
              />
            )}
          </main>
        </>
      )}
    </section>
  );
};
