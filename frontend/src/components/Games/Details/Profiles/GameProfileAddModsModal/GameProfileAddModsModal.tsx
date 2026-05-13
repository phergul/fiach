import { useEffect, useMemo, useState } from 'react';

import { Search, X } from 'lucide-react';

import type { Mod } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';

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
  const trimmedSearchQuery = searchQuery.trim().toLowerCase();
  const filteredMods = useMemo(() => {
    if (trimmedSearchQuery === '') {
      return availableMods;
    }

    return availableMods.filter((mod) => {
      return (
        mod.Name.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourcePath.toLowerCase().includes(trimmedSearchQuery)
      );
    });
  }, [availableMods, trimmedSearchQuery]);
  const selectedCount = selectedModIDs.size;

  useEffect(() => {
    if (!isOpen) {
      setSearchQuery('');
      setSelectedModIDs(new Set());
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

  if (!isOpen) {
    return null;
  }

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

  const handleClose = () => {
    if (isBusy) {
      return;
    }

    onClose();
  };

  const handleAddMods = async () => {
    if (selectedCount === 0) {
      return;
    }

    await onAddMods(Array.from(selectedModIDs));
    onClose();
  };

  return (
    <div className="game-profile-add-mods-modal" role="presentation">
      <div className="game-profile-add-mods-modal-backdrop" onClick={handleClose} aria-hidden="true" />

      <section
        className="game-profile-add-mods-modal-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="game-profile-add-mods-modal-title"
      >
        <header className="game-profile-add-mods-modal-header">
          <div className="game-profile-add-mods-modal-heading">
            <h2 className="game-profile-add-mods-modal-title" id="game-profile-add-mods-modal-title">
              Add Mods to Profile
            </h2>
            <p className="game-profile-add-mods-modal-summary">
              {availableMods.length} available for {profileName} · {selectedCount} selected
            </p>
          </div>

          <button
            className="game-profile-add-mods-modal-close"
            disabled={isBusy}
            onClick={handleClose}
            title="Close add mods"
            type="button"
          >
            <X className="game-profile-add-mods-modal-icon" aria-hidden="true" />
          </button>
        </header>

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

        <div className="game-profile-add-mods-modal-body">
          {isGameModsLoading && (
            <StateBlock className="game-profile-add-mods-modal-state" message="Loading imported mods..." />
          )}

          {!isGameModsLoading && availableMods.length === 0 && (
            <StateBlock className="game-profile-add-mods-modal-state" message="No available mods to add." />
          )}

          {!isGameModsLoading && availableMods.length > 0 && filteredMods.length === 0 && (
            <StateBlock className="game-profile-add-mods-modal-state" message="No available mods match this search." />
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
                      <span className="game-profile-add-mods-modal-option-control" aria-hidden="true" />
                      <span className="game-profile-add-mods-modal-option-copy">
                        <span className="game-profile-add-mods-modal-option-name">{mod.Name}</span>
                      </span>
                    </label>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        <footer className="game-profile-add-mods-modal-footer">
          <button
            className="game-profile-add-mods-modal-add-button"
            disabled={isBusy || selectedCount === 0}
            onClick={handleAddMods}
            type="button"
          >
            Add Mods
          </button>
          <button
            className="game-profile-add-mods-modal-cancel-button"
            disabled={isBusy}
            onClick={handleClose}
            type="button"
          >
            Cancel
          </button>
        </footer>
      </section>
    </div>
  );
};
