import { useGameSearch } from '../../../hooks/useGameSearch';
import { useStoredGames } from '../../../hooks/useStoredGames';
import { GameGrid } from '../GameGrid/GameGrid';
import { GameSearch } from '../GameSearch/GameSearch';

import './GameLibrary.scss';

export const GameLibrary = () => {
  const { games, isLoading, isScanning, loadError, refreshGames, retryLoadGames } = useStoredGames();
  const { filteredGames, searchQuery, setSearchQuery } = useGameSearch(games);

  const hasGames = games.length > 0;
  const hasSearchQuery = searchQuery.trim().length > 0;
  const isInitialLoading = isLoading && !hasGames;
  const hasLoadError = loadError !== null && !hasGames;
  const hasNoSearchResults = hasGames && hasSearchQuery && filteredGames.length === 0;
  const hasEmptyLibrary = !isInitialLoading && !hasLoadError && !hasGames;

  return (
    <section className="game-library" aria-labelledby="game-library-title">
      <div className="game-library-header">
        <div className="game-library-heading">
          <h1 className="game-library-title" id="game-library-title">
            Library
          </h1>
          <p className="game-library-count">{games.length} games</p>
        </div>

        <div className="game-library-toolbar">
          <GameSearch searchQuery={searchQuery} onSearchQueryChange={setSearchQuery} />
          <button
            className="game-library-rescan"
            disabled={isScanning}
            onClick={refreshGames}
            type="button"
          >
            Rescan
          </button>
        </div>
      </div>

      {isInitialLoading && <p className="game-library-state">Loading games...</p>}

      {hasLoadError && (
        <div className="game-library-state">
          <p className="game-library-state-title">Could not load games.</p>
          <p className="game-library-state-message">{loadError}</p>
          <button className="game-library-state-action" onClick={retryLoadGames} type="button">
            Retry
          </button>
        </div>
      )}

      {hasEmptyLibrary && (
        <div className="game-library-state">
          <p className="game-library-state-title">No games found.</p>
          <button className="game-library-state-action" onClick={refreshGames} type="button">
            Scan Steam
          </button>
        </div>
      )}

      {hasNoSearchResults && (
        <p className="game-library-state">No games match "{searchQuery}".</p>
      )}

      {filteredGames.length > 0 && <GameGrid games={filteredGames} />}
    </section>
  );
};
