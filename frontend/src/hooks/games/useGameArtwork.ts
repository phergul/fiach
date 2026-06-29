import { useState } from 'react';

export type GameArtworkType = 'banner' | 'hero' | 'logo';

export const useGameArtwork = (appID: string, artworkType: GameArtworkType = 'banner') => {
  const [failedArtworkSource, setFailedArtworkSource] = useState('');

  const trimmedAppID = appID.trim();
  const artworkSource = /^\d+$/.test(trimmedAppID)
    ? `/artwork/steam/${trimmedAppID}/${artworkType}`
    : '';
  const visibleArtworkSource = artworkSource === failedArtworkSource ? '' : artworkSource;

  return {
    artworkSource: visibleArtworkSource,
    handleArtworkError: () => {
      if (artworkSource !== '') {
        setFailedArtworkSource(artworkSource);
      }
    },
  };
};
