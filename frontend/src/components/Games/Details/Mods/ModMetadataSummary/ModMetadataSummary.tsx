import type { Mod } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ModMetadataSummary.scss';

export interface ModMetadataSummaryItem {
  label: string;
  value: string;
  tone?: 'default' | 'version' | 'count';
}

interface ModMetadataSummaryProps {
  items: ModMetadataSummaryItem[];
}

export const formatModSourceType = (sourceType: ModSourceType) => {
  return sourceType === ModSourceType.ModSourceTypeArchive ? 'Archive' : 'Folder';
};

const metadataValue = (value: string | null | undefined) => {
  const trimmedValue = value?.trim() ?? '';
  return trimmedValue === '' ? null : trimmedValue;
};

const numberFormatter = new Intl.NumberFormat(undefined);

export const formatModMetadataCount = (value: number | null | undefined, noun: string) => {
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

export const buildModMetadataSummaryItems = (mod: Mod): ModMetadataSummaryItem[] => {
  const items: ModMetadataSummaryItem[] = [];
  const version = metadataValue(mod.Metadata?.Version.Effective);
  const author = metadataValue(mod.Metadata?.Author.Effective);

  if (version !== null) {
    items.push({
      label: 'Version',
      value: version,
      tone: 'version',
    });
  }
  if (author !== null) {
    items.push({
      label: 'Author',
      value: author,
    });
  }

  items.push(
    {
      label: 'Source',
      value: formatModSourceType(mod.SourceType),
    },
    {
      label: 'Files',
      value: formatModMetadataCount(mod.FileCount, 'file'),
      tone: 'count',
    },
    {
      label: 'Folders',
      value: formatModMetadataCount(mod.DirectoryCount, 'folder'),
      tone: 'count',
    },
    {
      label: 'Size',
      value: formatModMetadataBytes(mod.TotalSizeBytes),
      tone: 'count',
    },
  );

  return items;
};

export const ModMetadataSummary = ({ items }: ModMetadataSummaryProps) => {
  return (
    <dl className="mod-metadata-summary">
      {items.map((item) => (
        <div
          className={[
            'mod-metadata-summary-item',
            item.tone === undefined ? undefined : `mod-metadata-summary-item-${item.tone}`,
          ]
            .filter(Boolean)
            .join(' ')}
          key={item.label}
        >
          <dt className="mod-metadata-summary-label">{item.label}</dt>
          <dd className="mod-metadata-summary-value">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
};
