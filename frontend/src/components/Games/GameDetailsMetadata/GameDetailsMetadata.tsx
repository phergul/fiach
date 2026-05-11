import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameDetailsMetadata.scss';

interface GameDetailsMetadataProps {
  game: StoredGame;
}

const getSteamAppID = (game: StoredGame) => {
  if (game.Source !== 'steam' || game.SourceID.trim() === '') {
    return 'Not linked';
  }

  return game.SourceID;
};

export const GameDetailsMetadata = ({ game }: GameDetailsMetadataProps) => {
  const metadataItems = [
    { label: 'Steam App ID', value: getSteamAppID(game) },
    { label: 'Availability', value: game.Available ? 'Available' : 'Unavailable' },
  ];

  return (
    <dl className="game-details-metadata" aria-label="Game metadata">
      {metadataItems.map((item) => (
        <div className="game-details-metadata-item" key={item.label}>
          <dt className="game-details-metadata-label">{item.label}</dt>
          <dd className="game-details-metadata-value">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
};
