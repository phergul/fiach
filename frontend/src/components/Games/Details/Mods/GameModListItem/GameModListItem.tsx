import { Trash2 } from 'lucide-react';

import type { Mod } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import {
  buildModMetadataSummaryItems,
  ModMetadataSummary,
} from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';

import './GameModListItem.scss';

interface GameModListItemProps {
  isBusy: boolean;
  mod: Mod;
  onDeleteMod: (mod: Mod) => void;
}

const sourceLabel = (mod: Mod) => mod.OriginalSourceName ?? mod.OriginalSourcePath;

export const GameModListItem = ({ isBusy, mod, onDeleteMod }: GameModListItemProps) => {
  return (
    <li className="game-mod-list-item">
      <div className="game-mod-list-item-copy">
        <span className="game-mod-list-item-name">{mod.Name}</span>
        <ModMetadataSummary items={buildModMetadataSummaryItems(mod)} />
        <span className="game-mod-list-item-source">{sourceLabel(mod)}</span>
        <span className="game-mod-list-item-path">{mod.SourcePath}</span>
      </div>

      <div className="game-mod-list-item-actions">
        <button
          aria-label={`Delete ${mod.Name}`}
          className="game-mod-list-item-icon-button game-mod-list-item-icon-button-danger"
          disabled={isBusy}
          onClick={() => onDeleteMod(mod)}
          title="Delete mod"
          type="button"
        >
          <Trash2 className="game-mod-list-item-icon" aria-hidden="true" />
        </button>
      </div>
    </li>
  );
};
