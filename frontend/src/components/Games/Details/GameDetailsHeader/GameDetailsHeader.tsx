import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameDetailsHeader.scss';

interface GameDetailsHeaderProps {
  game: StoredGame;
  logoArtworkSource: string;
  onLogoArtworkError: () => void;
}

export const GameDetailsHeader = ({
  game,
  logoArtworkSource,
  onLogoArtworkError,
}: GameDetailsHeaderProps) => {
  return (
    <div className="game-details-header">
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
          onError={onLogoArtworkError}
        />
      )}
    </div>
  );
};
