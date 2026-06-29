import { useEffect, useMemo, useRef, useState } from 'react';

import { Archive, FolderOpen, Plus, Search, Upload } from 'lucide-react';

import {
  DeleteMod,
  GetModDeleteSummary,
  RenameTag,
  UpdateModDetails,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import type {
  Mod,
  ModDeleteSummary,
  TagColor,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { useToast } from '@components/Common/Toast/Toast';
import { GameModListHeader } from '@components/Games/Details/Mods/GameModListHeader/GameModListHeader';
import { GameModListItem } from '@components/Games/Details/Mods/GameModListItem/GameModListItem';
import { ModTagFilter } from '@components/Games/Details/Mods/ModTags/ModTagFilter/ModTagFilter';
import {
  ModMetadataSidePanel,
  type ModMetadataSaveInput,
} from '@components/Games/Details/Mods/ModMetadataSidePanel/ModMetadataSidePanel';
import type { UseGameModsResult } from '@hooks';
import { useClickOutside } from '@hooks';
import { getErrorMessage } from '@utils';

import './GameModsSection.scss';

interface GameModsSectionProps {
  isFileDropTargetActive?: boolean;
  isImportDisabled?: boolean;
  isUpdateDisabled?: boolean;
  modManager: UseGameModsResult;
  onModDeleted: () => Promise<void> | void;
  onImportArchive: () => void;
  onImportFolder: () => void;
  onUpdateArchiveMod: (mod: Mod) => void;
  onUpdateFolderMod: (mod: Mod) => void;
}

const deleteSummaryMessage = (mod: Mod | null, summary: ModDeleteSummary | null) => {
  if (mod === null) {
    return '';
  }
  if (summary === null) {
    return `Preparing to delete "${mod.Name}"...`;
  }

  const profileMessage =
    summary.ProfileUsageCount === 0
      ? 'It is not assigned to any profiles.'
      : `It will be removed from ${summary.ProfileUsageCount} ${
          summary.ProfileUsageCount === 1 ? 'profile' : 'profiles'
        }.`;
  const appliedMessage = summary.IsInAppliedProfile
    ? ' This mod is part of the currently applied profile.'
    : '';

  return `Delete "${summary.ModName}" and its managed files? ${profileMessage}${appliedMessage}`;
};

export const GameModsSection = ({
  isFileDropTargetActive = false,
  isImportDisabled = false,
  isUpdateDisabled = false,
  modManager,
  onModDeleted,
  onImportArchive,
  onImportFolder,
  onUpdateArchiveMod,
  onUpdateFolderMod,
}: GameModsSectionProps) => {
  const { addErrorToast, addToast } = useToast();
  const { gameTags, isLoading, loadError, mods, refreshMods } = modManager;
  const [deleteCandidate, setDeleteCandidate] = useState<Mod | null>(null);
  const [deleteSummary, setDeleteSummary] = useState<ModDeleteSummary | null>(null);
  const [editingMetadataModID, setEditingMetadataModID] = useState<number | null>(null);
  const [metadataSaveError, setMetadataSaveError] = useState<string | null>(null);
  const [isDeleteSummaryLoading, setIsDeleteSummaryLoading] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isSavingMetadata, setIsSavingMetadata] = useState(false);
  const [isImportMenuOpen, setIsImportMenuOpen] = useState(false);
  const importMenuAnchorRef = useRef<HTMLDivElement>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTagIDs, setSelectedTagIDs] = useState<number[]>([]);
  const trimmedSearchQuery = searchQuery.trim().toLowerCase();
  const filteredMods = useMemo(() => {
    return mods.filter((mod) => {
      const matchesSearch =
        trimmedSearchQuery === '' ||
        mod.Name.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourcePath.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourceType.toLowerCase().includes(trimmedSearchQuery) ||
        mod.OriginalSourcePath.toLowerCase().includes(trimmedSearchQuery) ||
        (mod.OriginalSourceName ?? '').toLowerCase().includes(trimmedSearchQuery) ||
        (mod.Metadata?.Version.Effective ?? '').toLowerCase().includes(trimmedSearchQuery) ||
        (mod.Metadata?.Author.Effective ?? '').toLowerCase().includes(trimmedSearchQuery) ||
        (mod.Metadata?.SourceURL.Effective ?? '').toLowerCase().includes(trimmedSearchQuery) ||
        (mod.Metadata?.Notes ?? '').toLowerCase().includes(trimmedSearchQuery) ||
        mod.Tags.some((tag) => tag.Name.toLowerCase().includes(trimmedSearchQuery));
      const matchesTags = selectedTagIDs.every((tagID) => mod.Tags.some((tag) => tag.ID === tagID));
      return matchesSearch && matchesTags;
    });
  }, [mods, selectedTagIDs, trimmedSearchQuery]);

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
  const isRowBusy = isDeleteBusy || isSavingMetadata || isUpdateDisabled;
  useClickOutside(
    importMenuAnchorRef,
    () => setIsImportMenuOpen(false),
    isImportMenuOpen && !isImportDisabled,
  );
  const selectedMetadataMod =
    editingMetadataModID === null
      ? null
      : (mods.find((mod) => mod.ID === editingMetadataModID) ?? null);

  useEffect(() => {
    const availableTagIDs = new Set(mods.flatMap((mod) => mod.Tags.map((tag) => tag.ID)));
    setSelectedTagIDs((currentTagIDs) =>
      currentTagIDs.filter((tagID) => availableTagIDs.has(tagID)),
    );
  }, [mods]);

  const openMetadataEditor = (mod: Mod) => {
    if (isRowBusy) {
      return;
    }

    setEditingMetadataModID(mod.ID);
    setMetadataSaveError(null);
  };

  const closeMetadataEditor = () => {
    if (isSavingMetadata) {
      return;
    }

    setEditingMetadataModID(null);
    setMetadataSaveError(null);
  };

  const saveModMetadata = async (input: ModMetadataSaveInput) => {
    const currentMod = mods.find((mod) => mod.ID === input.modID);
    if (currentMod === undefined) {
      return;
    }

    setIsSavingMetadata(true);
    setMetadataSaveError(null);

    try {
      await UpdateModDetails({
        ModID: input.modID,
        Name: input.name.trim(),
        Metadata: input.metadata,
        TagIDs: input.tags.flatMap((tag) => (tag.ID === null ? [] : [tag.ID])),
        NewTags: input.tags.flatMap((tag) =>
          tag.ID === null ? [{ Name: tag.Name, Color: tag.Color }] : [],
        ),
      });
      addToast({
        message: 'Mod metadata saved.',
        tone: 'success',
      });
      await refreshMods();
      setEditingMetadataModID(null);
    } catch (error) {
      const message = getErrorMessage(error);
      setMetadataSaveError(message);
      addErrorToast(error);
    } finally {
      setIsSavingMetadata(false);
    }
  };

  const renameTag = async (tagID: number, name: string, color: TagColor) => {
    try {
      const tag = await RenameTag(tagID, name, color);
      await refreshMods();
      return tag;
    } catch (error) {
      addErrorToast(error);
      throw error;
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
      addErrorToast(error);
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
        addErrorToast(refreshError);
      }
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsDeleting(false);
    }
  };

  return (
    <section
      className="game-mods-section"
      aria-label="Imported mods"
      {...(isFileDropTargetActive ? { 'data-file-drop-target': '' } : {})}
    >
      {isFileDropTargetActive && (
        <div className="game-mods-section-drop-overlay" aria-hidden="true">
          <Upload className="game-mods-section-drop-overlay-icon" />
        </div>
      )}
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

        <ModTagFilter
          candidateMods={mods}
          onChange={setSelectedTagIDs}
          selectedTagIDs={selectedTagIDs}
        />

        <div className="game-mods-section-import-anchor" ref={importMenuAnchorRef}>
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
        <StateBlock
          className="game-mods-section-state"
          title="Could not load mods."
          message={loadError}
        >
          <button className="game-mods-section-button" onClick={refreshMods} type="button">
            Retry
          </button>
        </StateBlock>
      )}

      {loadError === null && isLoading && (
        <StateBlock className="game-mods-section-empty" message="Loading imported mods..." />
      )}

      {loadError === null && !isLoading && mods.length === 0 && (
        <StateBlock
          className="game-mods-section-empty"
          message="Imported mods for this game will appear here."
        />
      )}

      {loadError === null && !isLoading && mods.length > 0 && filteredMods.length === 0 && (
        <StateBlock
          className="game-mods-section-empty"
          message="No imported mods match this search."
        />
      )}

      {loadError === null && !isLoading && filteredMods.length > 0 && (
        <div
          className={
            selectedMetadataMod === null
              ? 'game-mods-section-content'
              : 'game-mods-section-content game-mods-section-content-with-panel'
          }
        >
          <div className="game-mods-section-list-shell">
            <GameModListHeader />
            <ul className="game-mods-section-list">
              {filteredMods.map((mod) => (
                <GameModListItem
                  isBusy={isRowBusy}
                  isEditing={editingMetadataModID === mod.ID}
                  key={mod.ID}
                  mod={mod}
                  onDeleteMod={requestDeleteMod}
                  onEditMod={openMetadataEditor}
                  onUpdateArchiveMod={onUpdateArchiveMod}
                  onUpdateFolderMod={onUpdateFolderMod}
                />
              ))}
            </ul>
          </div>

          <ModMetadataSidePanel
            availableTags={gameTags}
            error={metadataSaveError}
            isBusy={isSavingMetadata}
            mod={selectedMetadataMod}
            onClose={closeMetadataEditor}
            onRenameTag={renameTag}
            onSave={saveModMetadata}
          />
        </div>
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
