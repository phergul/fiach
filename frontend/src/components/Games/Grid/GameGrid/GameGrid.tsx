import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { GameCard } from '@components/Games/Grid/GameCard/GameCard';

import './GameGrid.scss';

interface GameGridProps {
  games: StoredGame[];
}

export const GameGrid = ({ games }: GameGridProps) => {
  return (
    <div className="game-grid">
      {games.map((game) => (
        <GameCard game={game} key={game.ID} />
      ))}
    </div>
  );
};
