import { useEffect, useState } from 'react';

import { GetGameImage } from '../../bindings/github.com/phergul/mod-manager/internal/services/steamservice';
import { ImageType } from '../../bindings/github.com/phergul/mod-manager/internal/steam/models';

export const useGameArtwork = (appID: string) => {
  const [artworkSource, setArtworkSource] = useState('');

  useEffect(() => {
    let isMounted = true;

    const loadArtwork = async () => {
      const trimmedAppID = appID.trim();
      if (trimmedAppID === '') {
        setArtworkSource('');
        return;
      }

      setArtworkSource('');

      try {
        const imageData = await GetGameImage(trimmedAppID, ImageType.ImageTypeBanner);
        if (isMounted) {
          setArtworkSource(imageData);
        }
      } catch {
        if (isMounted) {
          setArtworkSource('');
        }
      }
    };

    loadArtwork();

    return () => {
      isMounted = false;
    };
  }, [appID]);

  return artworkSource;
};
