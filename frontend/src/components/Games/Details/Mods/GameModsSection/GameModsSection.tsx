import { useMemo, useState } from 'react';

import { Archive, FolderOpen, Plus, Search } from 'lucide-react';

import {
  DeleteMod,
  GetModDeleteSummary,
  RenameMod,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import type { Mod, ModDeleteSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { useToast } from '@components/Common/Toast/Toast';
import { GameModListItem } from '@components/Games/Details/Mods/GameModListItem/GameModListItem';
import type { UseGameModsResult } from '@hooks';
import { getErrorMessage } from '@utils';

import './GameModsSection.scss';

interface GameModsSectionProps {
  isImportDisabled?: boolean;
  modManager: UseGameModsResult;
  onModDeleted: () => Promise<void> | void;
  onImportArchive: () => void;
  onImportFolder: () => void;
}

const deleteSummaryMessage = (mod: Mod | null, summary: ModDeleteSummary | null) => {
  if (mod === null) {
    return '';
  }
  if (summary === null) {
    return `Preparing to delete "${mod.Name}"...`;
  }

  const profileMessage = summary.ProfileUsageCount === 0
    ? 'It is not assigned to any profiles.'
    : `It will be removed from ${summary.ProfileUsageCount} ${summary.ProfileUsageCount === 1 ? 'profile' : 'profiles'
    }.`;
  const appliedMessage = summary.IsInAppliedProfile
    ? ' This mod is part of the currently applied profile.'
    : '';

  return `Delete "${summary.ModName}" and its managed files? ${profileMessage}${appliedMessage}`;
};

export const GameModsSection = ({
  isImportDisabled = false,
  modManager,
  onModDeleted,
  onImportArchive,
  onImportFolder,
}: GameModsSectionProps) => {
  const { addToast } = useToast();
  const { isLoading, loadError, mods, refreshMods } = modManager;
  const [deleteCandidate, setDeleteCandidate] = useState<Mod | null>(null);
  const [deleteSummary, setDeleteSummary] = useState<ModDeleteSummary | null>(null);
  const [editingModID, setEditingModID] = useState<number | null>(null);
  const [editingModName, setEditingModName] = useState('');
  const [isDeleteSummaryLoading, setIsDeleteSummaryLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isRenaming, setIsRenaming] = useState(false);
  const [isImportMenuOpen, setIsImportMenuOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const trimmedSearchQuery = searchQuery.trim().toLowerCase();
  const filteredMods = useMemo(() => {
    if (trimmedSearchQuery === '') {
      return mods;
    }

    return mods.filter((mod) => {
      return (
        mod.Name.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourcePath.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourceType.toLowerCase().includes(trimmedSearchQuery) ||
        mod.OriginalSourcePath.toLowerCase().includes(trimmedSearchQuery) ||
        (mod.OriginalSourceName ?? '').toLowerCase().includes(trimmedSearchQuery)
      );
    });
  }, [mods, trimmedSearchQuery]);

  const importMenuItems = [
    {
      icon: FolderOpen,
      label: 'Folder',
      onSelect: () => {
        setIsImportMenuOpen(false);
        onImportFolder();
      },
    },
    {
      icon: Archive,
      label: 'ZIP Archive',
      onSelect: () => {
        setIsImportMenuOpen(false);
        onImportArchive();
      },
    },
  ];
  const isDeleteBusy = isDeleteSummaryLoading || isDeleting;
  const isRowBusy = isDeleteBusy || isRenaming;

  const startRenameMod = (mod: Mod) => {
    if (isRowBusy) {
      return;
    }

    setEditingModID(mod.ID);
    setEditingModName(mod.Name);
  };

  const cancelRenameMod = () => {
    if (isRenaming) {
      return;
    }

    setEditingModID(null);
    setEditingModName('');
  };

  const renameMod = async (modID: number) => {
    const trimmedName = editingModName.trim();
    if (trimmedName === '') {
      addToast({
        message: 'Mod name is required.',
        tone: 'error',
      });
      return;
    }

    setIsRenaming(true);

    try {
      await RenameMod(modID, trimmedName);
      setEditingModID(null);
      setEditingModName('');
      addToast({
        message: 'Mod renamed.',
        tone: 'success',
      });
      await refreshMods();
    } catch (error) {
      addToast({
        message: getErrorMessage(error),
        tone: 'error',
      });
    } finally {
      setIsRenaming(false);
    }
  };

  const requestDeleteMod = async (mod: Mod) => {
    setDeleteCandidate(mod);
    setDeleteSummary(null);
    setIsDeleteSummaryLoading(true);

    try {
      const summary = await GetModDeleteSummary(mod.ID);
      setDeleteSummary(summary);
    } catch (error) {
      setDeleteCandidate(null);
      addToast({
        message: getErrorMessage(error),
        tone: 'error',
      });
    } finally {
      setIsDeleteSummaryLoading(false);
    }
  };

  const cancelDeleteMod = () => {
    if (isDeleteBusy) {
      return;
    }

    setDeleteCandidate(null);
    setDeleteSummary(null);
  };

  const confirmDeleteMod = async () => {
    if (deleteCandidate === null || deleteSummary === null) {
      return;
    }

    setIsDeleting(true);

    try {
      await DeleteMod(deleteCandidate.ID);
      setDeleteCandidate(null);
      setDeleteSummary(null);
      addToast({
        message: 'Mod deleted.',
        tone: 'success',
      });

      try {
        await refreshMods();
        await onModDeleted();
      } catch (refreshError) {
        addToast({
          message: getErrorMessage(refreshError),
          tone: 'error',
        });
      }
    } catch (error) {
      addToast({
        message: getErrorMessage(error),
        tone: 'error',
      });
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <section className="game-mods-section" aria-label="Imported mods">
      <div className="game-mods-section-controls">
        <div className="game-mods-section-search">
          <Search className="game-mods-section-search-icon" aria-hidden="true" />
          <input
            className="game-mods-section-search-input"
            disabled={isLoading || mods.length === 0}
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder="Search mods..."
            type="search"
            value={searchQuery}
          />
        </div>

        <div className="game-mods-section-import-anchor">
          <button
            className="game-mods-section-import-button"
            disabled={isImportDisabled}
            onClick={() => setIsImportMenuOpen((currentValue) => !currentValue)}
            type="button"
            aria-expanded={isImportMenuOpen}
          >
            <Plus className="game-mods-section-button-icon" aria-hidden="true" />
            Import Mod
          </button>

          <DropdownMenu
            ariaLabel="Import mod"
            isOpen={isImportMenuOpen && !isImportDisabled}
            items={importMenuItems}
          />
        </div>
      </div>

      {loadError !== null && (
        <StateBlock className="game-mods-section-state" title="Could not load mods." message={loadError}>
          <button className="game-mods-section-button" onClick={refreshMods} type="button">
            Retry
          </button>
        </StateBlock>
      )}

      {loadError === null && isLoading && (
        <StateBlock className="game-mods-section-empty" message="Loading imported mods..." />
      )}

      {loadError === null && !isLoading && mods.length === 0 && (
        <StateBlock className="game-mods-section-empty" message="Imported mods for this game will appear here." />
      )}

      {loadError === null && !isLoading && mods.length > 0 && filteredMods.length === 0 && (
        <StateBlock className="game-mods-section-empty" message="No imported mods match this search." />
      )}

      {loadError === null && !isLoading && filteredMods.length > 0 && (
        <ul className="game-mods-section-list">
          {filteredMods.map((mod) => (
            <GameModListItem
              editingModName={editingModName}
              isBusy={isRowBusy}
              isEditing={editingModID === mod.ID}
              key={mod.ID}
              mod={mod}
              onCancelRename={cancelRenameMod}
              onDeleteMod={requestDeleteMod}
              onEditingModNameChange={setEditingModName}
              onRenameMod={renameMod}
              onStartRename={startRenameMod}
            />
          ))}
        </ul>
      )}

      <ConfirmDialog
        confirmLabel="Delete"
        isBusy={isDeleteBusy}
        isOpen={deleteCandidate !== null}
        message={deleteSummaryMessage(deleteCandidate, deleteSummary)}
        onCancel={cancelDeleteMod}
        onConfirm={confirmDeleteMod}
        title="Delete mod"
      />
    </section>
  );
};
