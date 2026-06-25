import type { CSSProperties } from 'react';

import { ArrowDown, ArrowUp, GripVertical, Trash2 } from 'lucide-react';

import { useSortable } from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';

import type {
  ProfileMod,
  Tag,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModTagList } from '@components/Games/Details/Mods/ModTags/ModTagList/ModTagList';

import './GameProfileAssignedModRow.scss';

interface GameProfileAssignedModRowProps {
  canMoveDown: boolean;
  canMoveUp: boolean;
  canReorder: boolean;
  isBusy: boolean;
  mod: ProfileMod;
  tags: Tag[];
  onMoveDown: () => void;
  onMoveUp: () => void;
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModRow = ({
  canMoveDown,
  canMoveUp,
  canReorder,
  isBusy,
  mod,
  tags,
  onMoveDown,
  onMoveUp,
  onRemoveMod,
  onSetModEnabled,
}: GameProfileAssignedModRowProps) => {
  const displayLoadOrder = mod.LoadOrder + 1;
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: mod.ModID,
    disabled: isBusy || !canReorder,
  });
  const style: CSSProperties = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  return (
    <li
      ref={setNodeRef}
      className={
        isDragging
          ? 'game-profile-assigned-mod-row game-profile-assigned-mod-row-dragging'
          : 'game-profile-assigned-mod-row'
      }
      style={style}
    >
      <div
        className="game-profile-assigned-mod-row-order"
        aria-label={`Load order ${displayLoadOrder}`}
      >
        {displayLoadOrder}
      </div>

      <div className="game-profile-assigned-mod-row-handle">
        <button
          aria-label={`Drag ${mod.Name}`}
          className="game-profile-assigned-mod-row-handle-button"
          disabled={isBusy || !canReorder}
          type="button"
          {...attributes}
          {...listeners}
        >
          <GripVertical className="game-profile-assigned-mod-row-handle-icon" aria-hidden="true" />
        </button>
      </div>

      <div className="game-profile-assigned-mod-row-main">
        <span className="game-profile-assigned-mod-row-name">{mod.Name}</span>
        <ModTagList tags={tags} />
      </div>

      <div className="game-profile-assigned-mod-row-actions">
        <div className="game-profile-assigned-mod-row-order-actions">
          <button
            aria-label={`Move ${mod.Name} up`}
            className="game-profile-assigned-mod-row-icon-button"
            disabled={isBusy || !canReorder || !canMoveUp}
            onClick={onMoveUp}
            title="Move mod up"
            type="button"
          >
            <ArrowUp className="game-profile-assigned-mod-row-icon" aria-hidden="true" />
          </button>

          <button
            aria-label={`Move ${mod.Name} down`}
            className="game-profile-assigned-mod-row-icon-button"
            disabled={isBusy || !canReorder || !canMoveDown}
            onClick={onMoveDown}
            title="Move mod down"
            type="button"
          >
            <ArrowDown className="game-profile-assigned-mod-row-icon" aria-hidden="true" />
          </button>
        </div>

        <label
          className="game-profile-assigned-mod-row-toggle"
          title={mod.Enabled ? 'Disable mod' : 'Enable mod'}
        >
          <input
            aria-label={`${mod.Enabled ? 'Disable' : 'Enable'} ${mod.Name}`}
            checked={mod.Enabled}
            disabled={isBusy}
            onChange={(event) => onSetModEnabled(mod.ModID, event.target.checked)}
            type="checkbox"
          />
          <span className="game-profile-assigned-mod-row-toggle-control" aria-hidden="true" />
        </label>

        <button
          aria-label={`Remove ${mod.Name} from profile`}
          className="game-profile-assigned-mod-row-icon-button game-profile-assigned-mod-row-icon-button-danger"
          disabled={isBusy}
          onClick={() => onRemoveMod(mod.ModID)}
          title="Remove mod from profile"
          type="button"
        >
          <Trash2 className="game-profile-assigned-mod-row-icon" aria-hidden="true" />
        </button>
      </div>
    </li>
  );
};
