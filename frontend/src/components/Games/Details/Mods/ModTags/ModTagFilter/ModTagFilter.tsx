import { useMemo, useRef, useState } from 'react';

import { ListFilter, Search } from 'lucide-react';

import type { Mod, Tag } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModTagChip } from '@components/Games/Details/Mods/ModTags/ModTagChip/ModTagChip';
import { useClickOutside } from '@hooks';

import './ModTagFilter.scss';

interface ModTagFilterProps {
  candidateMods: Mod[];
  popoverPlacement?: 'above' | 'below';
  selectedTagIDs: number[];
  variant?: 'default' | 'profile-footer';
  onChange: (tagIDs: number[]) => void;
}

export const ModTagFilter = ({
  candidateMods,
  popoverPlacement = 'below',
  selectedTagIDs,
  variant = 'default',
  onChange,
}: ModTagFilterProps) => {
  const [isOpen, setIsOpen] = useState(false);
  const [query, setQuery] = useState('');
  const controlRef = useRef<HTMLDivElement>(null);
  useClickOutside(controlRef, () => setIsOpen(false), isOpen);
  const tags = useMemo(() => {
    const tagsByID = new Map<number, Tag>();
    candidateMods.forEach((mod) => mod.Tags.forEach((tag) => tagsByID.set(tag.ID, tag)));
    return Array.from(tagsByID.values()).sort((left, right) => left.Name.localeCompare(right.Name));
  }, [candidateMods]);
  const filteredTags = tags.filter((tag) =>
    tag.Name.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase()),
  );

  const toggleTag = (tagID: number) => {
    onChange(
      selectedTagIDs.includes(tagID)
        ? selectedTagIDs.filter((selectedID) => selectedID !== tagID)
        : [...selectedTagIDs, tagID],
    );
  };

  const searchField = (
    <label className="mod-tag-filter-search">
      <Search className="mod-tag-filter-search-icon" aria-hidden="true" />
      <input
        className="mod-tag-filter-search-input"
        onChange={(event) => setQuery(event.target.value)}
        placeholder="Find tags"
        type="search"
        value={query}
      />
    </label>
  );

  return (
    <div className={`mod-tag-filter mod-tag-filter-${popoverPlacement} mod-tag-filter-${variant}`}>
      <div className="mod-tag-filter-control" ref={controlRef}>
        <button
          aria-expanded={isOpen}
          className={
            selectedTagIDs.length > 0
              ? 'mod-tag-filter-button mod-tag-filter-button-active'
              : 'mod-tag-filter-button'
          }
          disabled={tags.length === 0}
          onClick={() => setIsOpen((currentValue) => !currentValue)}
          title={tags.length === 0 ? 'No tags available' : 'Filter by tags'}
          type="button"
        >
          <ListFilter className="mod-tag-filter-icon" aria-hidden="true" />
          {tags.length === 0 ? (
            'No tags'
          ) : variant === 'profile-footer' ? (
            <>
              <span>Tags</span>
              {selectedTagIDs.length > 0 && (
                <span className="mod-tag-filter-count">({selectedTagIDs.length})</span>
              )}
            </>
          ) : (
            `Tags${selectedTagIDs.length > 0 ? ` (${selectedTagIDs.length})` : ''}`
          )}
        </button>

        {isOpen && tags.length > 0 && (
          <div className="mod-tag-filter-popover">
            {variant !== 'profile-footer' && searchField}
            <div className="mod-tag-filter-options">
              {filteredTags.map((tag) => (
                <label className="dropdown-menu-checkbox-option" key={tag.ID}>
                  <input
                    checked={selectedTagIDs.includes(tag.ID)}
                    onChange={() => toggleTag(tag.ID)}
                    type="checkbox"
                  />
                  <span className="dropdown-menu-checkbox-control" aria-hidden="true" />
                  <ModTagChip color={tag.Color} name={tag.Name} />
                </label>
              ))}
            </div>
            {variant === 'profile-footer' && searchField}
          </div>
        )}
      </div>
    </div>
  );
};
