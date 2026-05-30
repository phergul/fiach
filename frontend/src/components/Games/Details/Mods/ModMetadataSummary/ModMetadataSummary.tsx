import type { Mod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

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

const numberFormatter = new Intl.NumberFormat(undefined);

const formatCount = (value: number | null | undefined, noun: string) => {
  if (value === null || value === undefined) {
    return 'Unavailable';
  }

  return `${numberFormatter.format(value)} ${value === 1 ? noun : `${noun}s`}`;
};

export const formatModMetadataBytes = (value: number | null | undefined) => {
  if (value === null || value === undefined) {
    return 'Unavailable';
  }
  if (value < 1024) {
    return `${numberFormatter.format(value)} B`;
  }

  const units = ['KB', 'MB', 'GB', 'TB'];
  let size = value / 1024;
  let unitIndex = 0;
  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  return `${new Intl.NumberFormat(undefined, {
    maximumFractionDigits: size >= 10 ? 1 : 2,
  }).format(size)} ${units[unitIndex]}`;
};

export const buildModMetadataSummaryItems = (mod: Mod): ModMetadataSummaryItem[] => [
  {
    label: 'Source',
    value: sourceTypeLabel(mod.SourceType),
  },
  {
    label: 'Files',
    value: formatCount(mod.FileCount, 'file'),
  },
  {
    label: 'Folders',
    value: formatCount(mod.DirectoryCount, 'folder'),
  },
  {
    label: 'Size',
    value: formatModMetadataBytes(mod.TotalSizeBytes),
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
