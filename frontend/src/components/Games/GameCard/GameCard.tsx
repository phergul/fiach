import type { StoredGame } from '../../../../bindings/github.com/phergul/mod-manager/internal/storage/models';

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
  return (
    <article className="game-card">
      <div className="game-card-artwork">
        <span className="game-card-artwork-title">{game.Name}</span>
        <span className="game-card-source" aria-label={getSourceLabel(game.Source)} title={getSourceLabel(game.Source)}>
          {getSourceInitial(game.Source)}
        </span>
      </div>
      <h2 className="game-card-title">{game.Name}</h2>
    </article>
  );
};
