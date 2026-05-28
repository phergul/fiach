import type { Mod } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { ModSourceType } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';

import './ModMetadataSummary.scss';

export interface ModMetadataSummaryItem {
  label: string;
  value: string;
}

interface ModMetadataSummaryProps {
  items: ModMetadataSummaryItem[];
}

const dateFormatter = new Intl.DateTimeFormat(undefined, {
  dateStyle: 'medium',
  timeStyle: 'short',
});

const formatTimestamp = (value: string) => {
  const trimmedValue = value.trim();
  if (trimmedValue === '') {
    return '-';
  }

  const normalizedValue = trimmedValue.includes('T') ? trimmedValue : `${trimmedValue.replace(' ', 'T')}Z`;
  const date = new Date(normalizedValue);
  if (Number.isNaN(date.getTime())) {
    return trimmedValue;
  }

  return dateFormatter.format(date);
};

const sourceTypeLabel = (sourceType: ModSourceType) => {
  return sourceType === ModSourceType.ModSourceTypeArchive ? 'Archive' : 'Folder';
};

export const buildModMetadataSummaryItems = (mod: Mod): ModMetadataSummaryItem[] => [
  {
    label: 'Source',
    value: sourceTypeLabel(mod.SourceType),
  },
  {
    label: 'Imported',
    value: formatTimestamp(mod.CreatedAt),
  },
  {
    label: 'Updated',
    value: formatTimestamp(mod.UpdatedAt),
  },
];

export const ModMetadataSummary = ({ items }: ModMetadataSummaryProps) => {
  return (
    <dl className="mod-metadata-summary">
      {items.map((item) => (
        <div className="mod-metadata-summary-item" key={item.label}>
          <dt className="mod-metadata-summary-label">{item.label}</dt>
          <dd className="mod-metadata-summary-value">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
};
