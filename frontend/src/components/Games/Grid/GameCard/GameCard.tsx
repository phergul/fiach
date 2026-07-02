import { Link } from 'react-router-dom';

import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useGameArtwork } from '@hooks';
import steamLogo from '@assets/steam.svg';

import './GameCard.scss';

interface GameCardProps {
  game: StoredGame;
}

const getSourceLabel = (source: string) => {
  return source === 'steam' ? 'Steam game' : 'Custom game';
};

export const GameCard = ({ game }: GameCardProps) => {
  const { artworkSource, handleArtworkError } = useGameArtwork(
    game.Source === 'steam' && game.SourceID ? game.SourceID : '',
  );

  return (
    <article className="game-card">
      <Link
        className="game-card-link"
        to={`/library/${game.ID}`}
        aria-label={`View ${game.Name} details`}
      >
        <div className="game-card-artwork">
          {artworkSource !== '' && (
            <img
              className="game-card-image"
              src={artworkSource}
              alt={`${game.Name} artwork`}
              onError={handleArtworkError}
            />
          )}
          <span
            className="game-card-source"
            aria-label={getSourceLabel(game.Source)}
            title={getSourceLabel(game.Source)}
          >
            {game.Source === 'steam' ? (
              <img className="game-card-source-image" src={steamLogo} alt="" />
            ) : (
              'M'
            )}
          </span>
        </div>
        <h2 className="game-card-title">{game.Name}</h2>
      </Link>
    </article>
  );
};
