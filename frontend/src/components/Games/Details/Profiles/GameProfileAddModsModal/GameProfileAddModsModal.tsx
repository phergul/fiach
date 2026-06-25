import { useEffect, useMemo, useState } from 'react';

import { Search } from 'lucide-react';

import type { Mod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Modal } from '@components/Common/Modal/Modal';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import {
  buildModMetadataSummaryItems,
  ModMetadataSummary,
} from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';
import { ModTagFilter } from '@components/Games/Details/Mods/ModTags/ModTagFilter/ModTagFilter';
import { ModTagList } from '@components/Games/Details/Mods/ModTags/ModTagList/ModTagList';

import './GameProfileAddModsModal.scss';

interface GameProfileAddModsModalProps {
  availableMods: Mod[];
  isBusy: boolean;
  isGameModsLoading: boolean;
  isOpen: boolean;
  profileName: string;
  onAddMods: (modIDs: number[]) => Promise<void> | void;
  onClose: () => void;
}

export const GameProfileAddModsModal = ({
  availableMods,
  isBusy,
  isGameModsLoading,
  isOpen,
  profileName,
  onAddMods,
  onClose,
}: GameProfileAddModsModalProps) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedModIDs, setSelectedModIDs] = useState<Set<number>>(() => new Set());
  const [selectedTagIDs, setSelectedTagIDs] = useState<number[]>([]);
  const trimmedSearchQuery = searchQuery.trim().toLowerCase();
  const filteredMods = useMemo(() => {
    return availableMods.filter((mod) => {
      const matchesSearch =
        trimmedSearchQuery === '' ||
        mod.Name.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourcePath.toLowerCase().includes(trimmedSearchQuery) ||
        mod.Tags.some((tag) => tag.Name.toLowerCase().includes(trimmedSearchQuery));
      const matchesTags = selectedTagIDs.every((tagID) => mod.Tags.some((tag) => tag.ID === tagID));
      return matchesSearch && matchesTags;
    });
  }, [availableMods, selectedTagIDs, trimmedSearchQuery]);
  const selectedCount = selectedModIDs.size;

  useEffect(() => {
    if (!isOpen) {
      setSearchQuery('');
      setSelectedModIDs(new Set());
      setSelectedTagIDs([]);
    }
  }, [isOpen]);

  useEffect(() => {
    setSelectedModIDs((currentSelectedModIDs) => {
      const availableModIDs = new Set(availableMods.map((mod) => mod.ID));
      const nextSelectedModIDs = new Set<number>();
      currentSelectedModIDs.forEach((modID) => {
        if (availableModIDs.has(modID)) {
          nextSelectedModIDs.add(modID);
        }
      });
      return nextSelectedModIDs;
    });
  }, [availableMods]);

  const toggleSelectedMod = (modID: number) => {
    setSelectedModIDs((currentSelectedModIDs) => {
      const nextSelectedModIDs = new Set(currentSelectedModIDs);
      if (nextSelectedModIDs.has(modID)) {
        nextSelectedModIDs.delete(modID);
      } else {
        nextSelectedModIDs.add(modID);
      }
      return nextSelectedModIDs;
    });
  };

  const handleAddMods = async () => {
    if (selectedCount === 0) {
      return;
    }

    await onAddMods(Array.from(selectedModIDs));
    onClose();
  };

  return (
    <Modal
      bodyClassName="game-profile-add-mods-modal-body"
      closeTitle="Close add mods"
      description={`${availableMods.length} available for ${profileName} · ${selectedCount} selected`}
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-profile-add-mods-modal-title"
      onClose={onClose}
      panelClassName="game-profile-add-mods-modal-panel"
      size="lg"
      title="Add Mods to Profile"
      footer={
        <div className="game-profile-add-mods-modal-footer">
          <button
            className="game-profile-add-mods-modal-cancel-button"
            disabled={isBusy}
            onClick={onClose}
            type="button"
          >
            Cancel
          </button>
          <button
            className="game-profile-add-mods-modal-add-button button-main"
            disabled={isBusy || selectedCount === 0}
            onClick={handleAddMods}
            type="button"
          >
            Add Mods
          </button>
        </div>
      }
    >
      <>
        <div className="game-profile-add-mods-modal-controls">
          <div className="game-profile-add-mods-modal-search">
            <Search className="game-profile-add-mods-modal-search-icon" aria-hidden="true" />
            <input
              className="game-profile-add-mods-modal-search-input"
              disabled={isBusy || isGameModsLoading || availableMods.length === 0}
              onChange={(event) => setSearchQuery(event.target.value)}
              placeholder="Search available mods..."
              type="search"
              value={searchQuery}
            />
          </div>
          <ModTagFilter
            candidateMods={availableMods}
            onChange={setSelectedTagIDs}
            selectedTagIDs={selectedTagIDs}
          />
        </div>

        <div className="game-profile-add-mods-modal-list-body">
          {isGameModsLoading && (
            <StateBlock
              className="game-profile-add-mods-modal-state"
              message="Loading imported mods..."
            />
          )}

          {!isGameModsLoading && availableMods.length === 0 && (
            <StateBlock
              className="game-profile-add-mods-modal-state"
              message="No available mods to add."
            />
          )}

          {!isGameModsLoading && availableMods.length > 0 && filteredMods.length === 0 && (
            <StateBlock
              className="game-profile-add-mods-modal-state"
              message="No available mods match this search."
            />
          )}

          {!isGameModsLoading && filteredMods.length > 0 && (
            <ul className="game-profile-add-mods-modal-list">
              {filteredMods.map((mod) => {
                const isSelected = selectedModIDs.has(mod.ID);

                return (
                  <li className="game-profile-add-mods-modal-list-item" key={mod.ID}>
                    <label className="game-profile-add-mods-modal-option">
                      <input
                        checked={isSelected}
                        disabled={isBusy}
                        onChange={() => toggleSelectedMod(mod.ID)}
                        type="checkbox"
                      />
                      <span
                        className="game-profile-add-mods-modal-option-control"
                        aria-hidden="true"
                      />
                      <span className="game-profile-add-mods-modal-option-copy">
                        <span className="game-profile-add-mods-modal-option-name">{mod.Name}</span>
                        <ModMetadataSummary items={buildModMetadataSummaryItems(mod)} />
                        <ModTagList tags={mod.Tags} />
                      </span>
                    </label>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
      </>
    </Modal>
  );
};
