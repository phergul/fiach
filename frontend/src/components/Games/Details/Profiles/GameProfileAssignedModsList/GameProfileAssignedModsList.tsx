import { useEffect, useMemo, useState } from 'react';

import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  type DragEndEvent,
  useSensor,
  useSensors,
} from '@dnd-kit/core';
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable';

import type { ProfileMod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { GameProfileAssignedModRow } from '@components/Games/Details/Profiles/GameProfileAssignedModRow/GameProfileAssignedModRow';

import './GameProfileAssignedModsList.scss';

interface GameProfileAssignedModsListProps {
  canReorder: boolean;
  isBusy: boolean;
  mods: ProfileMod[];
  onMoveMod: (modID: number, direction: -1 | 1) => void;
  onReorderMods: (orderedModIDs: number[]) => Promise<void> | void;
  onRemoveMod: (modID: number) => void;
  onSetModEnabled: (modID: number, enabled: boolean) => void;
}

export const GameProfileAssignedModsList = ({
  canReorder,
  isBusy,
  mods,
  onMoveMod,
  onReorderMods,
  onRemoveMod,
  onSetModEnabled,
}: GameProfileAssignedModsListProps) => {
  const [orderedMods, setOrderedMods] = useState(mods);
  const activeModIDs = useMemo(() => orderedMods.map((mod) => mod.ModID), [orderedMods]);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  useEffect(() => {
    setOrderedMods(mods);
  }, [mods]);

  const handleDragEnd = (event: DragEndEvent) => {
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
    <DndContext collisionDetection={closestCenter} onDragEnd={handleDragEnd} sensors={sensors}>
      <SortableContext items={activeModIDs} strategy={verticalListSortingStrategy}>
        <ul
          className={'game-profile-assigned-mods-list'}
          aria-label="Assigned profile mods"
        >
          {orderedMods.map((mod, index) => (
            <GameProfileAssignedModRow
              key={mod.ModID}
              canMoveDown={index < orderedMods.length - 1}
              canMoveUp={index > 0}
              canReorder={canReorder}
              isBusy={isBusy}
              mod={mod}
              onMoveDown={() => onMoveMod(mod.ModID, 1)}
              onMoveUp={() => onMoveMod(mod.ModID, -1)}
              onRemoveMod={onRemoveMod}
              onSetModEnabled={onSetModEnabled}
            />
          ))}
        </ul>
      </SortableContext>
    </DndContext>
  );
};
