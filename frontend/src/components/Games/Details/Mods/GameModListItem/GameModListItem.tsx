import { useState } from 'react';

import { Archive, FolderOpen, Pencil, RefreshCw, Trash2 } from 'lucide-react';

import type { Mod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import {
  buildModMetadataSummaryItems,
  ModMetadataSummary,
} from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';
import { ModTagList } from '@components/Games/Details/Mods/ModTags/ModTagList/ModTagList';

import './GameModListItem.scss';

interface GameModListItemProps {
  isBusy: boolean;
  mod: Mod;
  onDeleteMod: (mod: Mod) => void;
  onEditMod: (mod: Mod) => void;
  onUpdateArchiveMod: (mod: Mod) => void;
  onUpdateFolderMod: (mod: Mod) => void;
}

const sourceLabel = (mod: Mod) => mod.OriginalSourceName ?? mod.OriginalSourcePath;

export const GameModListItem = ({
  isBusy,
  mod,
  onDeleteMod,
  onEditMod,
  onUpdateArchiveMod,
  onUpdateFolderMod,
}: GameModListItemProps) => {
  const [isUpdateMenuOpen, setIsUpdateMenuOpen] = useState(false);

  const updateMenuItems = [
    {
      icon: FolderOpen,
      label: 'Folder',
      onSelect: () => {
        setIsUpdateMenuOpen(false);
        onUpdateFolderMod(mod);
      },
    },
    {
      icon: Archive,
      label: 'ZIP Archive',
      onSelect: () => {
        setIsUpdateMenuOpen(false);
        onUpdateArchiveMod(mod);
      },
    },
  ];

  return (
    <li className="game-mod-list-item">
      <div className="game-mod-list-item-copy">
        <span className="game-mod-list-item-name">{mod.Name}</span>
        <ModMetadataSummary items={buildModMetadataSummaryItems(mod)} />
        <ModTagList tags={mod.Tags} />
        <span className="game-mod-list-item-source">{sourceLabel(mod)}</span>
      </div>

      <div className="game-mod-list-item-actions">
        <div className="game-mod-list-item-update-anchor">
          <button
            aria-expanded={isUpdateMenuOpen}
            aria-label={`Update ${mod.Name}`}
            className="game-mod-list-item-icon-button"
            disabled={isBusy}
            onClick={() => setIsUpdateMenuOpen((currentValue) => !currentValue)}
            title="Update mod"
            type="button"
          >
            <RefreshCw className="game-mod-list-item-icon" aria-hidden="true" />
          </button>

          <DropdownMenu
            ariaLabel={`Update ${mod.Name}`}
            isOpen={isUpdateMenuOpen && !isBusy}
            items={updateMenuItems}
          />
        </div>
        <button
          aria-label={`Edit ${mod.Name} metadata`}
          className="game-mod-list-item-icon-button"
          disabled={isBusy}
          onClick={() => onEditMod(mod)}
          title="Edit mod metadata"
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
      </div>
    </li>
  );
};
