import { useMemo, useState } from 'react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

export const useGameSearch = (games: StoredGame[]) => {
  const [searchQuery, setSearchQuery] = useState('');

  const filteredGames = useMemo(() => {
    const normalizedQuery = searchQuery.trim().toLowerCase();
    if (normalizedQuery === '') {
      return games;
    }

    return games.filter((game) => {
      return (
        game.Name.toLowerCase().includes(normalizedQuery) ||
        game.InstallPath.toLowerCase().includes(normalizedQuery)
      );
    });
  }, [games, searchQuery]);

  return {
    filteredGames,
    searchQuery,
    setSearchQuery,
  };
};
