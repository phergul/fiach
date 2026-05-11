import { useState } from 'react';

import { Link, useParams } from 'react-router-dom';

import { ImageType } from '@bindings/github.com/phergul/mod-manager/internal/steam/models';
import { GameDetailsMetadata } from '@components/Games/GameDetailsMetadata/GameDetailsMetadata';
import { GameModsSection } from '@components/Games/GameModsSection/GameModsSection';
import { GameProfilesSection } from '@components/Games/GameProfilesSection/GameProfilesSection';
import { useGameArtwork, useStoredGames } from '@hooks';

import './GameDetails.scss';

type GameDetailsTab = 'mods' | 'profiles';

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

export const GameDetails = () => {
  const { gameId } = useParams();
  const [activeTab, setActiveTab] = useState<GameDetailsTab>('mods');
  const { games, isLoading, isScanning, loadError, retryLoadGames } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const heroArtworkSource = useGameArtwork(
    game?.Source === 'steam' ? game.SourceID : '',
    ImageType.ImageTypeHero,
  );
  const logoArtworkSource = useGameArtwork(
    game?.Source === 'steam' ? game.SourceID : '',
    ImageType.ImageTypeLogo,
  );
  const isWaitingForGame = (isLoading || isScanning) && game === undefined;
  const hasLoadError = loadError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;

  return (
    <section
      className={heroArtworkSource === '' ? 'game-details' : 'game-details game-details-with-backdrop'}
      aria-label="Game details"
    >
      <div className="game-details-toolbar">
        <Link className="game-details-back-link" to="/library">
          Back
        </Link>
      </div>

      {isWaitingForGame && <p className="game-details-state">Loading game...</p>}

      {hasLoadError && (
        <div className="game-details-state">
          <p className="game-details-state-title">Could not load game.</p>
          <p className="game-details-state-message">{loadError}</p>
          <div className="game-details-state-actions">
            <button className="game-details-state-action" onClick={retryLoadGames} type="button">
              Retry
            </button>
            <Link className="game-details-state-link" to="/library">
              Return to library
            </Link>
          </div>
        </div>
      )}

      {hasNotFound && (
        <div className="game-details-state">
          <p className="game-details-state-title">Game not found.</p>
          <p className="game-details-state-message">
            This game is not currently available in the library.
          </p>
        </div>
      )}

      {game !== undefined && (
        <>
          <div className="game-details-header">
            {heroArtworkSource !== '' && (
              <div className="game-details-backdrop" aria-hidden="true">
                <img className="game-details-backdrop-image" src={heroArtworkSource} alt="" />
              </div>
            )}

            <div className="game-details-heading">
              <h1 className="game-details-title" id="game-details-title">
                {game.Name}
              </h1>
              <div className="game-details-install-path">
                <span className="game-details-install-path-label">Install path</span>
                <span className="game-details-install-path-value">{game.InstallPath}</span>
              </div>
            </div>

            {logoArtworkSource !== '' && (
              <img
                className="game-details-logo"
                src={logoArtworkSource}
                alt={`${game.Name} logo`}
              />
            )}
          </div>

          <GameDetailsMetadata game={game} />

          <div className="game-details-tabs" role="tablist" aria-label="Game detail sections">
            <button
              className={
                activeTab === 'mods'
                  ? 'game-details-tab game-details-tab-active'
                  : 'game-details-tab'
              }
              onClick={() => setActiveTab('mods')}
              role="tab"
              type="button"
              aria-selected={activeTab === 'mods'}
            >
              Imported mods
            </button>
            <button
              className={
                activeTab === 'profiles'
                  ? 'game-details-tab game-details-tab-active'
                  : 'game-details-tab'
              }
              onClick={() => setActiveTab('profiles')}
              role="tab"
              type="button"
              aria-selected={activeTab === 'profiles'}
            >
              Profiles
            </button>
          </div>

          {activeTab === 'mods' ? <GameModsSection /> : <GameProfilesSection />}
        </>
      )}
    </section>
  );
};
