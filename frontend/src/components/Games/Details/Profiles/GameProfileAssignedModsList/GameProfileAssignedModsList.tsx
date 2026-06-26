import { useEffect, useMemo, useState } from 'react';

import {
  closestCenter,
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  type DragEndEvent,
  type DragStartEvent,
  type Modifier,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';

import type {
  ProfileMod,
  Tag,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { GameProfileAssignedModRow } from '@components/Games/Details/Profiles/GameProfileAssignedModRow/GameProfileAssignedModRow';

import './GameProfileAssignedModsList.scss';

const restrictToVerticalDragBounds: Modifier = ({
  containerNodeRect,
  draggingNodeRect,
  scrollableAncestorRects,
  transform,
}) => {
  if (!draggingNodeRect) {
    return { ...transform, x: 0 };
  }

  let y = transform.y;
  const projectedTop = draggingNodeRect.top + y;
  const projectedBottom = draggingNodeRect.bottom + y;
  const bounds: Array<{ top: number; bottom: number }> = [];

  if (containerNodeRect) {
    bounds.push({
      top: containerNodeRect.top,
      bottom: containerNodeRect.bottom,
    });
  }

  const scrollableRect = scrollableAncestorRects[0];
  if (scrollableRect) {
    bounds.push({
      top: scrollableRect.top,
      bottom: scrollableRect.bottom,
    });
  }

  if (bounds.length > 0) {
    const top = Math.max(...bounds.map((bound) => bound.top));
    const bottom = Math.min(...bounds.map((bound) => bound.bottom));

    if (projectedTop < top) {
      y = top - draggingNodeRect.top;
    } else if (projectedBottom > bottom) {
      y = bottom - draggingNodeRect.bottom;
    }
  }

  return { ...transform, x: 0, y };
};

interface GameProfileAssignedModsListProps {
  canReorder: boolean;
  isBusy: boolean;
  mods: ProfileMod[];
  tagsByModID: Record<number, Tag[]>;
  onMoveMod: (modID: number, direction: -1 | 1) => void;
  onReorderMods: (orderedModIDs: number[]) => Promise<void> | void;
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModsList = ({
  canReorder,
  isBusy,
  mods,
  tagsByModID,
  onMoveMod,
  onReorderMods,
  onRemoveMod,
  onSetModEnabled,
}: GameProfileAssignedModsListProps) => {
  const [orderedMods, setOrderedMods] = useState(mods);
  const [activeModID, setActiveModID] = useState<number | null>(null);
  const [activeRowWidth, setActiveRowWidth] = useState<number | undefined>();
  const activeModIDs = useMemo(() => orderedMods.map((mod) => mod.ModID), [orderedMods]);
  const activeMod = useMemo(
    () => orderedMods.find((mod) => mod.ModID === activeModID),
    [activeModID, orderedMods],
  );
  const activeModIndex = activeModID === null ? -1 : activeModIDs.indexOf(activeModID);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  useEffect(() => {
    setOrderedMods(mods);
  }, [mods]);

  const clearActiveDrag = () => {
    setActiveModID(null);
    setActiveRowWidth(undefined);
  };

  const handleDragStart = (event: DragStartEvent) => {
    setActiveModID(Number(event.active.id));
    setActiveRowWidth(event.active.rect.current?.initial?.width ?? undefined);
  };

  const handleDragEnd = (event: DragEndEvent) => {
    clearActiveDrag();

    if (!canReorder || isBusy || event.over === null || event.active.id === event.over.id) {
      return;
    }

    const oldIndex = activeModIDs.indexOf(Number(event.active.id));
    const newIndex = activeModIDs.indexOf(Number(event.over.id));
    if (oldIndex < 0 || newIndex < 0 || oldIndex === newIndex) {
      return;
    }

    const reorderedMods = arrayMove(orderedMods, oldIndex, newIndex);
    setOrderedMods(reorderedMods);
    onReorderMods(reorderedMods.map((mod) => mod.ModID));
  };

  return (
    <DndContext
      autoScroll={{ threshold: { x: 0, y: 0.2 } }}
      collisionDetection={closestCenter}
      modifiers={[restrictToVerticalDragBounds]}
      onDragCancel={clearActiveDrag}
      onDragEnd={handleDragEnd}
      onDragStart={handleDragStart}
      sensors={sensors}
    >
      <SortableContext items={activeModIDs} strategy={verticalListSortingStrategy}>
        <ul className={'game-profile-assigned-mods-list'} aria-label="Assigned profile mods">
          {orderedMods.map((mod, index) => (
            <GameProfileAssignedModRow
              key={mod.ModID}
              canMoveDown={index < orderedMods.length - 1}
              canMoveUp={index > 0}
              canReorder={canReorder}
              isBusy={isBusy}
              mod={mod}
              tags={tagsByModID[mod.ModID] ?? []}
              onMoveDown={() => onMoveMod(mod.ModID, 1)}
              onMoveUp={() => onMoveMod(mod.ModID, -1)}
              onRemoveMod={onRemoveMod}
              onSetModEnabled={onSetModEnabled}
            />
          ))}
        </ul>
      </SortableContext>

      <DragOverlay dropAnimation={null}>
        {activeMod ? (
          <div
            className="game-profile-assigned-mods-list-overlay"
            style={{ width: activeRowWidth }}
          >
            <GameProfileAssignedModRow
              canMoveDown={activeModIndex < orderedMods.length - 1}
              canMoveUp={activeModIndex > 0}
              canReorder={canReorder}
              dragOverlay
              isBusy={isBusy}
              mod={activeMod}
              tags={tagsByModID[activeMod.ModID] ?? []}
              onMoveDown={() => onMoveMod(activeMod.ModID, 1)}
              onMoveUp={() => onMoveMod(activeMod.ModID, -1)}
              onRemoveMod={onRemoveMod}
              onSetModEnabled={onSetModEnabled}
            />
          </div>
        ) : null}
      </DragOverlay>
    </DndContext>
  );
};
