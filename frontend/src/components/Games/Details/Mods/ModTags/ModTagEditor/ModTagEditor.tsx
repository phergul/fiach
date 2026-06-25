import { useMemo, useState } from 'react';

import { Check, Plus, X } from 'lucide-react';

import {
  TagColor,
  type Tag,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModTagChip } from '@components/Games/Details/Mods/ModTags/ModTagChip/ModTagChip';

import './ModTagEditor.scss';

export interface ModTagSelection {
  ID: number | null;
  Name: string;
  Color: TagColor;
}

interface ModTagEditorProps {
  availableTags: Tag[];
  isBusy: boolean;
  selectedTags: ModTagSelection[];
  onChange: (tags: ModTagSelection[]) => void;
  onRenameTag?: (tagID: number, name: string, color: TagColor) => Promise<Tag>;
}

const tagColors = [
  TagColor.TagColorRed,
  TagColor.TagColorOrange,
  TagColor.TagColorYellow,
  TagColor.TagColorGreen,
  TagColor.TagColorTeal,
  TagColor.TagColorBlue,
  TagColor.TagColorPurple,
  TagColor.TagColorPink,
];

const normalizedTagName = (name: string) => name.trim().toLocaleLowerCase();

export const ModTagEditor = ({
  availableTags,
  isBusy,
  selectedTags,
  onChange,
  onRenameTag,
}: ModTagEditorProps) => {
  const [isAdding, setIsAdding] = useState(false);
  const [query, setQuery] = useState('');
  const [newTagColor, setNewTagColor] = useState<TagColor | null>(null);
  const [editingTag, setEditingTag] = useState<ModTagSelection | null>(null);
  const [editingName, setEditingName] = useState('');
  const [editingColor, setEditingColor] = useState<TagColor>(TagColor.TagColorRed);
  const [isRenaming, setIsRenaming] = useState(false);
  const selectedIDs = useMemo(
    () => new Set(selectedTags.flatMap((tag) => (tag.ID === null ? [] : [tag.ID]))),
    [selectedTags],
  );
  const suggestions = useMemo(() => {
    const normalizedQuery = normalizedTagName(query);
    return availableTags.filter((tag) => {
      return (
        !selectedIDs.has(tag.ID) &&
        (normalizedQuery === '' || normalizedTagName(tag.Name).includes(normalizedQuery))
      );
    });
  }, [availableTags, query, selectedIDs]);
  const canCreate =
    query.trim() !== '' &&
    newTagColor !== null &&
    !availableTags.some((tag) => normalizedTagName(tag.Name) === normalizedTagName(query)) &&
    !selectedTags.some((tag) => normalizedTagName(tag.Name) === normalizedTagName(query));

  const closeAdd = () => {
    setIsAdding(false);
    setQuery('');
    setNewTagColor(null);
  };

  const addExistingTag = (tag: Tag) => {
    onChange([
      ...selectedTags,
      {
        ID: tag.ID,
        Name: tag.Name,
        Color: tag.Color,
      },
    ]);
    closeAdd();
  };

  const addNewTag = () => {
    if (!canCreate || newTagColor === null) {
      return;
    }

    onChange([
      ...selectedTags,
      {
        ID: null,
        Name: query.trim(),
        Color: newTagColor,
      },
    ]);
    closeAdd();
  };

  const startEditing = (tag: ModTagSelection) => {
    setEditingTag(tag);
    setEditingName(tag.Name);
    setEditingColor(tag.Color);
  };

  const saveEditingTag = async () => {
    if (editingTag === null || editingName.trim() === '') {
      return;
    }

    if (editingTag.ID === null || onRenameTag === undefined) {
      onChange(
        selectedTags.map((tag) =>
          tag === editingTag ? { ...tag, Name: editingName.trim(), Color: editingColor } : tag,
        ),
      );
      setEditingTag(null);
      return;
    }

    setIsRenaming(true);
    try {
      const renamed = await onRenameTag(editingTag.ID, editingName.trim(), editingColor);
      const nextTags = selectedTags
        .map((tag) =>
          tag.ID === editingTag.ID
            ? { ID: renamed.ID, Name: renamed.Name, Color: renamed.Color }
            : tag,
        )
        .filter(
          (tag, index, allTags) =>
            tag.ID === null || allTags.findIndex((candidate) => candidate.ID === tag.ID) === index,
        );
      onChange(nextTags);
      setEditingTag(null);
    } catch {
      // The caller owns user-facing error reporting.
    } finally {
      setIsRenaming(false);
    }
  };

  return (
    <div className="mod-tag-editor">
      <div className="mod-tag-editor-list">
        {selectedTags.map((tag, index) => (
          <ModTagChip
            color={tag.Color}
            key={tag.ID ?? `new-${normalizedTagName(tag.Name)}-${index}`}
            name={tag.Name}
            onClick={() => startEditing(tag)}
            onRemove={() => onChange(selectedTags.filter((_, tagIndex) => tagIndex !== index))}
          />
        ))}
        <button
          aria-label="Add tag"
          className="mod-tag-editor-add-button"
          disabled={isBusy || selectedTags.length >= 20}
          onClick={() => setIsAdding(true)}
          title="Add tag"
          type="button"
        >
          <Plus className="mod-tag-editor-icon" aria-hidden="true" />
        </button>
      </div>

      {editingTag !== null && (
        <div className="mod-tag-editor-inline">
          <div className="mod-tag-editor-inline-row">
            <input
              className="mod-tag-editor-input"
              disabled={isBusy || isRenaming}
              maxLength={50}
              onChange={(event) => setEditingName(event.target.value)}
              type="text"
              value={editingName}
            />
            <button
              aria-label="Save tag"
              className="mod-tag-editor-inline-button"
              disabled={isBusy || isRenaming || editingName.trim() === ''}
              onClick={saveEditingTag}
              type="button"
            >
              <Check className="mod-tag-editor-icon" aria-hidden="true" />
            </button>
            <button
              aria-label="Cancel tag edit"
              className="mod-tag-editor-inline-button"
              disabled={isRenaming}
              onClick={() => setEditingTag(null)}
              type="button"
            >
              <X className="mod-tag-editor-icon" aria-hidden="true" />
            </button>
          </div>
          <div className="mod-tag-editor-colors" aria-label="Tag color">
            {tagColors.map((color) => (
              <button
                aria-label={`Use ${color}`}
                aria-pressed={editingColor === color}
                className={`mod-tag-editor-color mod-tag-editor-color-${color}`}
                disabled={isBusy || isRenaming}
                key={color}
                onClick={() => setEditingColor(color)}
                type="button"
              />
            ))}
          </div>
        </div>
      )}

      {isAdding && (
        <div className="mod-tag-editor-add-panel">
          <div className="mod-tag-editor-add-row">
            <input
              autoFocus
              className="mod-tag-editor-input"
              disabled={isBusy}
              maxLength={50}
              onChange={(event) => setQuery(event.target.value)}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.preventDefault();
                  const exactSuggestion = suggestions.find(
                    (tag) => normalizedTagName(tag.Name) === normalizedTagName(query),
                  );
                  if (exactSuggestion !== undefined) {
                    addExistingTag(exactSuggestion);
                  } else {
                    addNewTag();
                  }
                }
              }}
              placeholder="Find or create a tag"
              type="text"
              value={query}
            />
            <button
              aria-label="Close tag picker"
              className="mod-tag-editor-inline-button"
              onClick={closeAdd}
              type="button"
            >
              <X className="mod-tag-editor-icon" aria-hidden="true" />
            </button>
          </div>

          {suggestions.length > 0 && (
            <div className="mod-tag-editor-suggestions">
              {suggestions.map((tag) => (
                <button
                  className="mod-tag-editor-suggestion"
                  key={tag.ID}
                  onClick={() => addExistingTag(tag)}
                  type="button"
                >
                  <ModTagChip color={tag.Color} name={tag.Name} />
                </button>
              ))}
            </div>
          )}

          <div className="mod-tag-editor-create">
            <div className="mod-tag-editor-colors" aria-label="New tag color">
              {tagColors.map((color) => (
                <button
                  aria-label={`Use ${color}`}
                  aria-pressed={newTagColor === color}
                  className={`mod-tag-editor-color mod-tag-editor-color-${color}`}
                  disabled={isBusy}
                  key={color}
                  onClick={() => setNewTagColor(color)}
                  type="button"
                />
              ))}
            </div>
            <button
              className="mod-tag-editor-create-button"
              disabled={!canCreate || isBusy}
              onClick={addNewTag}
              type="button"
            >
              Create tag
            </button>
          </div>
        </div>
      )}
    </div>
  );
};
