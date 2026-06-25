import { X } from 'lucide-react';

import type { TagColor } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ModTagChip.scss';

interface ModTagChipProps {
  color: TagColor;
  name: string;
  onClick?: () => void;
  onRemove?: () => void;
}

export const ModTagChip = ({ color, name, onClick, onRemove }: ModTagChipProps) => {
  const content = <span className="mod-tag-chip-value">{name}</span>;

  return (
    <span className={`mod-tag-chip mod-tag-chip-${color}`}>
      {onClick === undefined ? (
        <span className="mod-tag-chip-content">{content}</span>
      ) : (
        <button
          className="mod-tag-chip-content mod-tag-chip-edit"
          onClick={onClick}
          title={`Edit ${name}`}
          type="button"
        >
          {content}
        </button>
      )}
      {onRemove !== undefined && (
        <button
          aria-label={`Remove ${name}`}
          className="mod-tag-chip-remove"
          onClick={onRemove}
          type="button"
        >
          <X className="mod-tag-chip-remove-icon" aria-hidden="true" />
        </button>
      )}
    </span>
  );
};
