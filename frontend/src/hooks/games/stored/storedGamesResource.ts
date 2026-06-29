import { GetStoredGames } from '@bindings/github.com/phergul/fiach/internal/services/gamesservice';
import type { StoredGame } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { createSingletonCachedResource } from '../../cache/createSingletonCachedResource';

export const storedGamesResource = createSingletonCachedResource<StoredGame[]>({
  emptyValue: [],
  fetch: () => GetStoredGames(),
});

export const fetchStoredGames = storedGamesResource.fetch;
export const preloadStoredGames = storedGamesResource.preload;
export const invalidateStoredGames = storedGamesResource.invalidate;
