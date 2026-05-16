import { Link } from 'react-router-dom';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { useGameArtwork } from '@hooks';

import './GameCard.scss';

interface GameCardProps {
  game: StoredGame;
}

const getSourceLabel = (source: string) => {
  return source === 'steam' ? 'Steam game' : 'Custom game';
};

const getSourceInitial = (source: string) => {
  return source === 'steam' ? 'S' : 'M';
};

export const GameCard = ({ game }: GameCardProps) => {
  const { artworkSource, handleArtworkError } = useGameArtwork(
    game.Source === 'steam' && game.SourceID ? game.SourceID : '',
  );

  return (
    <article className="game-card">
      <Link className="game-card-link" to={`/library/${game.ID}`} aria-label={`View ${game.Name} details`}>
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
            {getSourceInitial(game.Source)}
          </span>
        </div>
        <h2 className="game-card-title">{game.Name}</h2>
      </Link>
    </article>
  );
};
