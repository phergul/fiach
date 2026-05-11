import { CheckCircle2, CircleCheck, Package, Users } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameDetailsMetadata.scss';

interface GameDetailsMetadataProps {
  game: StoredGame;
}

export const GameDetailsMetadata = ({ game }: GameDetailsMetadataProps) => {
  const metadataItems = [
    { Icon: CheckCircle2, label: 'Available', value: game.Available ? 'Yes' : 'No' },
    { Icon: Package, label: 'Mods installed', value: '0' },
    { Icon: CircleCheck, label: 'Mods enabled', value: '0' },
    { Icon: Users, label: 'Profiles', value: '0' },
  ];

  return (
    <dl className="game-details-metadata" aria-label="Game metadata">
      {metadataItems.map((item) => (
        <div className="game-details-metadata-item" key={item.label}>
          <dt className="game-details-metadata-label">
            <item.Icon className="game-details-metadata-icon" aria-hidden="true" />
            <span>{item.label}</span>
          </dt>
          <dd className="game-details-metadata-value">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
};
