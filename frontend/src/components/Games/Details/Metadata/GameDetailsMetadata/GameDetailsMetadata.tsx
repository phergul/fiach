import { CheckCircle2, CircleSlash2, Database, Package, Users } from 'lucide-react';

import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';

import './GameDetailsMetadata.scss';

interface GameDetailsMetadataProps {
  game: StoredGame;
  isStorageUsageLoading: boolean;
  modCount: number;
  profileCount: number;
  storageUsedBytes: number | null;
}

const formatStorageUsage = (bytes: number | null, isLoading: boolean) => {
  if (isLoading || bytes === null) {
    return '-';
  }

  if (bytes < 1024) {
    return `${bytes} B`;
  }

  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes / 1024;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const formattedValue = value >= 100 ? value.toFixed(0) : value.toFixed(2);
  return `${formattedValue} ${units[unitIndex]}`;
};

export const GameDetailsMetadata = ({
  game,
  isStorageUsageLoading,
  modCount,
  profileCount,
  storageUsedBytes,
}: GameDetailsMetadataProps) => {
  const metadataItems = [
    { Icon: game.Available ? CheckCircle2 : CircleSlash2, label: 'Available', value: game.Available ? 'Yes' : 'No' },
    { Icon: Package, label: 'Mods installed', value: String(modCount) },
    { Icon: Database, label: 'Storage used', value: formatStorageUsage(storageUsedBytes, isStorageUsageLoading) },
    { Icon: Users, label: 'Profiles', value: String(profileCount) },
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
