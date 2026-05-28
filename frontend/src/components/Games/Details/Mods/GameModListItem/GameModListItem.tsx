import { Check, Pencil, Trash2, X } from 'lucide-react';

import type { Mod } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import {
  buildModMetadataSummaryItems,
  ModMetadataSummary,
} from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';

import './GameModListItem.scss';

interface GameModListItemProps {
  editingModName: string;
  isBusy: boolean;
  isEditing: boolean;
  mod: Mod;
  onCancelRename: () => void;
  onDeleteMod: (mod: Mod) => void;
  onEditingModNameChange: (name: string) => void;
  onRenameMod: (modID: number) => void;
  onStartRename: (mod: Mod) => void;
}

const sourceLabel = (mod: Mod) => mod.OriginalSourceName ?? mod.OriginalSourcePath;

export const GameModListItem = ({
  editingModName,
  isBusy,
  isEditing,
  mod,
  onCancelRename,
  onDeleteMod,
  onEditingModNameChange,
  onRenameMod,
  onStartRename,
}: GameModListItemProps) => {
  return (
    <li className="game-mod-list-item">
      <div className="game-mod-list-item-copy">
        {isEditing ? (
          <input
            aria-label={`Rename ${mod.Name}`}
            className="game-mod-list-item-input"
            disabled={isBusy}
            onChange={(event) => onEditingModNameChange(event.target.value)}
            type="text"
            value={editingModName}
          />
        ) : (
          <span className="game-mod-list-item-name">{mod.Name}</span>
        )}
        <ModMetadataSummary items={buildModMetadataSummaryItems(mod)} />
        <span className="game-mod-list-item-source">{sourceLabel(mod)}</span>
        <span className="game-mod-list-item-path">{mod.SourcePath}</span>
      </div>

      <div className="game-mod-list-item-actions">
        {isEditing ? (
          <>
            <button
              aria-label={`Save ${mod.Name} name`}
              className="game-mod-list-item-icon-button"
              disabled={isBusy}
              onClick={() => onRenameMod(mod.ID)}
              title="Save mod name"
              type="button"
            >
              <Check className="game-mod-list-item-icon" aria-hidden="true" />
            </button>
            <button
              aria-label="Cancel rename"
              className="game-mod-list-item-icon-button"
              disabled={isBusy}
              onClick={onCancelRename}
              title="Cancel rename"
              type="button"
            >
              <X className="game-mod-list-item-icon" aria-hidden="true" />
            </button>
          </>
        ) : (
          <>
            <button
              aria-label={`Rename ${mod.Name}`}
              className="game-mod-list-item-icon-button"
              disabled={isBusy}
              onClick={() => onStartRename(mod)}
              title="Rename mod"
              type="button"
            >
              <Pencil className="game-mod-list-item-icon" aria-hidden="true" />
            </button>
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
          </>
        )}
      </div>
    </li>
  );
};
