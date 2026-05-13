import { useMemo, useState } from 'react';

import { Search } from 'lucide-react';

import { StateBlock } from '@components/Common/StateBlock/StateBlock';
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
