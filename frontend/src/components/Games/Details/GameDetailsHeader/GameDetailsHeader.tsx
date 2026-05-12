import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameDetailsHeader.scss';

interface GameDetailsHeaderProps {
  game: StoredGame;
  heroArtworkSource: string;
  logoArtworkSource: string;
}

export const GameDetailsHeader = ({
  game,
  heroArtworkSource,
  logoArtworkSource,
}: GameDetailsHeaderProps) => {
  return (
    <div className="game-details-header">
      {heroArtworkSource !== '' && (
        <div className="game-details-header-backdrop" aria-hidden="true">
          <img className="game-details-header-backdrop-image" src={heroArtworkSource} alt="" />
        </div>
      )}

      <div className="game-details-header-heading">
        <h1 className="game-details-header-title" id="game-details-title">
          {game.Name}
        </h1>
      </div>

      {logoArtworkSource !== '' && (
        <img
          className="game-details-header-logo"
          src={logoArtworkSource}
          alt={`${game.Name} logo`}
        />
      )}
    </div>
  );
};
