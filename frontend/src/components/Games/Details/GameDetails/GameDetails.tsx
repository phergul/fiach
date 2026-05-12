import { useState } from 'react';

import { Link, useParams } from 'react-router-dom';

import { ImageType } from '@bindings/github.com/phergul/mod-manager/internal/steam/models';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { GameDetailsTabs, type GameDetailsTab } from '@components/Games/Details/GameDetailsTabs/GameDetailsTabs';
import { GameDetailsMetadata } from '@components/Games/Details/Metadata/GameDetailsMetadata/GameDetailsMetadata';
import { GameModsSection } from '@components/Games/Details/Mods/GameModsSection/GameModsSection';
import { GameProfilesSection } from '@components/Games/Details/Profiles/GameProfilesSection/GameProfilesSection';
import { useGameArtwork, useGameMods, useGameProfiles, useStoredGames } from '@hooks';

import './GameDetails.scss';

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
  const [activeTab, setActiveTab] = useState<GameDetailsTab>('profiles');
  const { games, isLoading, isScanning, loadError, retryLoadGames } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const gameModManager = useGameMods(game?.ID ?? null);
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
            heroArtworkSource={heroArtworkSource}
            logoArtworkSource={logoArtworkSource}
          />

          <GameDetailsMetadata
            game={game}
            modCount={gameModManager.mods.length}
            profileCount={profileManager.profiles.length}
            profileModsByProfileID={profileManager.profileModsByProfileID}
          />

          <GameDetailsTabs activeTab={activeTab} onActiveTabChange={setActiveTab} />

          {activeTab === 'mods' ? (
            <GameModsSection modManager={gameModManager} />
          ) : (
            <GameProfilesSection gameModManager={gameModManager} profileManager={profileManager} />
          )}
        </>
      )}
    </section>
  );
};
