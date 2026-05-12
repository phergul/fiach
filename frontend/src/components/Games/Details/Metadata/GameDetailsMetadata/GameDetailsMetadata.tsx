import { CircleSlash2, CheckCircle2, CircleCheck, Package, Users } from 'lucide-react';

import type { ProfileMod, StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';

import './GameDetailsMetadata.scss';

interface GameDetailsMetadataProps {
  game: StoredGame;
  modCount: number;
  profileCount: number;
  profileModsByProfileID: Record<number, ProfileMod[]>;
}

export const GameDetailsMetadata = ({ game, modCount, profileCount, profileModsByProfileID }: GameDetailsMetadataProps) => {
  const enabledProfileModCount = Object.values(profileModsByProfileID).reduce((enabledCount, profileMods) => {
    return enabledCount + profileMods.filter((profileMod) => profileMod.Enabled).length;
  }, 0);
  const metadataItems = [
    { Icon: game.Available ? CheckCircle2 : CircleSlash2, label: 'Available', value: game.Available ? 'Yes' : 'No' },
    { Icon: Package, label: 'Mods installed', value: String(modCount) },
    { Icon: CircleCheck, label: 'Mods enabled', value: String(enabledProfileModCount) },
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
