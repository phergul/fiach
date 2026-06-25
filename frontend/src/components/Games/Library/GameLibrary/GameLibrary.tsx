import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { GameGrid } from '@components/Games/Grid/GameGrid/GameGrid';
import { GameSearch } from '@components/Games/Library/GameSearch/GameSearch';
import { useGameSearch, useStoredGames } from '@hooks';

import './GameLibrary.scss';

export const GameLibrary = () => {
  const { games, isLoading, isScanning, loadError, refreshGames, retryLoadGames } =
    useStoredGames();
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

      {isInitialLoading && <StateBlock className="game-library-state" message="Loading games..." />}

      {hasLoadError && (
        <StateBlock
          className="game-library-state"
          title="Could not load games."
          message={loadError}
        >
          <button className="game-library-state-action" onClick={retryLoadGames} type="button">
            Retry
          </button>
        </StateBlock>
      )}

      {hasEmptyLibrary && (
        <StateBlock className="game-library-state" title="No games found.">
          <button className="game-library-state-action" onClick={refreshGames} type="button">
            Scan Steam
          </button>
        </StateBlock>
      )}

      {hasNoSearchResults && (
        <StateBlock className="game-library-state" message={`No games match "${searchQuery}".`} />
      )}

      {filteredGames.length > 0 && <GameGrid games={filteredGames} />}
    </section>
  );
};
