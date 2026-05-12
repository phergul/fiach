import { useMemo, useState } from 'react';

import { Search } from 'lucide-react';

import type { UseGameModsResult } from '@hooks';

import './GameModsSection.scss';

interface GameModsSectionProps {
  modManager: UseGameModsResult;
}

export const GameModsSection = ({ modManager }: GameModsSectionProps) => {
  const { isLoading, loadError, mods, refreshMods } = modManager;
  const [searchQuery, setSearchQuery] = useState('');
  const trimmedSearchQuery = searchQuery.trim().toLowerCase();
  const filteredMods = useMemo(() => {
    if (trimmedSearchQuery === '') {
      return mods;
    }

    return mods.filter((mod) => {
      return (
        mod.Name.toLowerCase().includes(trimmedSearchQuery) ||
        mod.SourcePath.toLowerCase().includes(trimmedSearchQuery)
      );
    });
  }, [mods, trimmedSearchQuery]);

  return (
    <section className="game-mods-section" aria-label="Imported mods">
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

      {loadError !== null && (
        <div className="game-mods-section-state">
          <p className="game-mods-section-state-title">Could not load mods.</p>
          <p className="game-mods-section-state-message">{loadError}</p>
          <button className="game-mods-section-button" onClick={refreshMods} type="button">
            Retry
          </button>
        </div>
      )}

      {loadError === null && isLoading && (
        <p className="game-mods-section-empty">Loading imported mods...</p>
      )}

      {loadError === null && !isLoading && mods.length === 0 && (
        <p className="game-mods-section-empty">Imported mods for this game will appear here.</p>
      )}

      {loadError === null && !isLoading && mods.length > 0 && filteredMods.length === 0 && (
        <p className="game-mods-section-empty">No imported mods match this search.</p>
      )}

      {loadError === null && !isLoading && filteredMods.length > 0 && (
        <ul className="game-mods-section-list">
          {filteredMods.map((mod) => (
            <li className="game-mods-section-list-item" key={mod.ID}>
              <div className="game-mods-section-list-item-copy">
                <span className="game-mods-section-list-item-name">{mod.Name}</span>
                <span className="game-mods-section-list-item-path">{mod.SourcePath}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};
