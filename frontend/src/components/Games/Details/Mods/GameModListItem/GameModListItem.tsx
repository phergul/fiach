import { useRef, useState } from 'react';

import { Archive, FolderOpen, Pencil, RefreshCw, Trash2 } from 'lucide-react';

import type { Mod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { useClickOutside } from '@hooks';
import {
  formatModMetadataBytes,
  formatModMetadataCount,
  formatModSourceType,
} from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';
import { ModTagList } from '@components/Games/Details/Mods/ModTags/ModTagList/ModTagList';

import './GameModListItem.scss';

interface GameModListItemProps {
  isBusy: boolean;
  isEditing: boolean;
  mod: Mod;
  onDeleteMod: (mod: Mod) => void;
  onEditMod: (mod: Mod) => void;
  onUpdateArchiveMod: (mod: Mod) => void;
  onUpdateFolderMod: (mod: Mod) => void;
}

const sourceLabel = (mod: Mod) => mod.OriginalSourceName ?? mod.OriginalSourcePath;

export const GameModListItem = ({
  isBusy,
  isEditing,
  mod,
  onDeleteMod,
  onEditMod,
  onUpdateArchiveMod,
  onUpdateFolderMod,
}: GameModListItemProps) => {
  const [isUpdateMenuOpen, setIsUpdateMenuOpen] = useState(false);
  const updateMenuAnchorRef = useRef<HTMLDivElement>(null);
  useClickOutside(
    updateMenuAnchorRef,
    () => setIsUpdateMenuOpen(false),
    isUpdateMenuOpen && !isBusy,
  );
  const version = mod.Metadata?.Version.Effective?.trim() ?? '';
  const author = mod.Metadata?.Author.Effective?.trim() ?? '';
  const identityMetadata = [
    version === '' ? null : `Version ${version}`,
    author === '' ? null : `by ${author}`,
  ].filter((value): value is string => value !== null);

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
    <li
      className={isEditing ? 'game-mod-list-item game-mod-list-item-editing' : 'game-mod-list-item'}
    >
      <div className="game-mod-list-item-identity">
        <span className="game-mod-list-item-name">{mod.Name}</span>
        {identityMetadata.length > 0 && (
          <span className="game-mod-list-item-identity-metadata">{identityMetadata.join(' ')}</span>
        )}
        <span className="game-mod-list-item-path" title={sourceLabel(mod)}>
          {sourceLabel(mod)}
        </span>
      </div>

      <div className="game-mod-list-item-tags">
        {mod.Tags.length > 0 ? (
          <ModTagList tags={mod.Tags} />
        ) : (
          <span className="game-mod-list-item-empty-value">No tags</span>
        )}
      </div>

      <div className="game-mod-list-item-metadata">
        <div className="game-mod-list-item-metadata-field">
          <span className="game-mod-list-item-metadata-label">Source</span>
          <span className="game-mod-list-item-metadata-value">
            {formatModSourceType(mod.SourceType)}
          </span>
        </div>
        <div className="game-mod-list-item-metadata-field game-mod-list-item-contents">
          <span className="game-mod-list-item-metadata-label">Contents</span>
          <span className="game-mod-list-item-metadata-value">
            {formatModMetadataCount(mod.FileCount, 'file')}
          </span>
          <span className="game-mod-list-item-metadata-secondary">
            {formatModMetadataCount(mod.DirectoryCount, 'folder')}
          </span>
        </div>
        <div className="game-mod-list-item-metadata-field">
          <span className="game-mod-list-item-metadata-label">Size</span>
          <span className="game-mod-list-item-metadata-value game-mod-list-item-numeric">
            {formatModMetadataBytes(mod.TotalSizeBytes)}
          </span>
        </div>
      </div>

      <div className="game-mod-list-item-actions">
        <div className="game-mod-list-item-update-anchor" ref={updateMenuAnchorRef}>
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
