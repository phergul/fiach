import type { Tag } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModTagChip } from '@components/Games/Details/Mods/ModTags/ModTagChip/ModTagChip';

import './ModTagList.scss';

interface ModTagListProps {
  tags: Tag[];
}

export const ModTagList = ({ tags }: ModTagListProps) => {
  if (tags.length === 0) {
    return null;
  }

  return (
    <div className="mod-tag-list" aria-label="Mod tags">
      {tags.map((tag) => (
        <ModTagChip color={tag.Color} key={tag.ID} name={tag.Name} />
      ))}
    </div>
  );
};
