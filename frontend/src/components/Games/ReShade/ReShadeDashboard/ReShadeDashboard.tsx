import { useState } from 'react';

import { ArrowLeft } from 'lucide-react';
import { Link, useParams } from 'react-router-dom';

import { Breadcrumbs } from '@components/Common/Breadcrumbs/Breadcrumbs';
import { InlineLoading } from '@components/Common/InlineLoading/InlineLoading';
import { useToast } from '@components/Common/Toast/Toast';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { ReShadePageHeader } from '@components/Games/ReShade/ReShadePageHeader/ReShadePageHeader';
import { ReShadeRecoveryPanel } from '@components/Games/ReShade/ReShadeRecoveryPanel/ReShadeRecoveryPanel';
import {
  ReShadeTargetTable,
  type ReShadeOperationSelection,
} from '@components/Games/ReShade/ReShadeTargetTable/ReShadeTargetTable';
import { ReShadeWizard } from '@components/Games/ReShade/ReShadeWizard/ReShadeWizard';
import { useGameArtwork, useGameReShade, useStoredGames } from '@hooks';

import './ReShadeDashboard.scss';

const parseGameID = (gameID: string | undefined) => {
  const parsed = Number(gameID);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null;
};

export const ReShadeDashboard = () => {
  const { gameId } = useParams();
  const { addErrorToast, addToast } = useToast();
  const {
    games,
    isLoading: isLoadingGames,
    isScanning,
    loadError: gamesError,
    retryLoadGames,
  } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game =
    parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const reShade = useGameReShade(game?.ID ?? null);
  const [operationSelection, setOperationSelection] = useState<ReShadeOperationSelection | null>(
    null,
  );
  const { artworkSource: heroArtworkSource, handleArtworkError: handleHeroArtworkError } =
    useGameArtwork(game?.Source === 'steam' && game.SourceID ? game.SourceID : '', 'hero');
  const { artworkSource: logoArtworkSource, handleArtworkError: handleLogoArtworkError } =
    useGameArtwork(game?.Source === 'steam' && game.SourceID ? game.SourceID : '', 'logo');
  const isWaitingForGame = (isLoadingGames || isScanning) && game === undefined;
  const hasLoadError = gamesError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;
  const gameDetailsPath = parsedGameID === null ? '/library' : `/library/${parsedGameID}`;
  const isRecoveryRequired = reShade.recovery?.required === true;

  const rollbackRecovery = async () => {
    try {
      const result = await reShade.rollbackRecovery();
      if (result !== null) {
        addToast({ message: result.message, tone: result.success ? 'success' : 'error' });
      }
    } catch (error) {
      addErrorToast(error);
    }
  };

  return (
    <section
      className={
        heroArtworkSource === ''
          ? 'reshade-dashboard'
          : 'reshade-dashboard reshade-dashboard-with-backdrop'
      }
      aria-label="ReShade management"
    >
      <div className="reshade-dashboard-toolbar">
        <Link className="reshade-dashboard-back-link" to={gameDetailsPath}>
          <ArrowLeft aria-hidden="true" />
          Back
        </Link>
      </div>

      {heroArtworkSource !== '' && (
        <div className="reshade-dashboard-backdrop" aria-hidden="true">
          <img
            className="reshade-dashboard-backdrop-image"
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

          <div className="reshade-dashboard-breadcrumbs">
            <Breadcrumbs
              items={[
                {
                  label: game.Name,
                },
                {
                  label: 'ReShade',
                },
              ]}
            />
          </div>

          <ReShadePageHeader
            installerStatus={reShade.installerStatus}
            isLoading={reShade.isLoading}
            onRefresh={() => void reShade.refresh(true)}
          />

          {reShade.loadError !== null && (
            <p className="reshade-dashboard-error">{reShade.loadError}</p>
          )}

          {isRecoveryRequired && reShade.recovery !== null && (
            <ReShadeRecoveryPanel
              isRollingBack={reShade.isRollingBack}
              onRollback={() => void rollbackRecovery()}
              recovery={reShade.recovery}
            />
          )}

          <main className="reshade-dashboard-content">
            {reShade.isRefreshing && (
              <InlineLoading
                className="reshade-dashboard-inline-loading"
                label="Refreshing ReShade targets..."
              />
            )}
            {operationSelection !== null && !isRecoveryRequired ? (
              <ReShadeWizard
                catalogue={reShade.catalogue}
                chainTargets={reShade.chainTargets}
                gameID={game.ID}
                onClose={() => setOperationSelection(null)}
                onRecoveryRequired={reShade.refresh}
                onRefresh={reShade.refresh}
                selection={operationSelection}
              />
            ) : reShade.isInitialLoading ? (
              <InlineLoading
                className="reshade-dashboard-inline-loading"
                label="Discovering ReShade targets..."
              />
            ) : reShade.loadError !== null ? (
              <GameDetailsState
                actionLabel="Retry"
                message={reShade.loadError}
                onAction={() => void reShade.refresh()}
                title="Could not load ReShade state."
              />
            ) : (
              <ReShadeTargetTable
                chainTargets={reShade.chainTargets}
                disabled={isRecoveryRequired}
                discovery={reShade.discovery}
                installerStatus={reShade.installerStatus}
                onStartOperation={setOperationSelection}
                targets={reShade.targets}
              />
            )}
          </main>
        </>
      )}
    </section>
  );
};
