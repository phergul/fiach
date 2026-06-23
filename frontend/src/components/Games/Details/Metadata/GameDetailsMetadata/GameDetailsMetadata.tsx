import { Database, Gauge, Package, Sparkles, Users } from 'lucide-react';

import {
  ReShadeDetectionStatus,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import type { UseGameOptiScalerResult, UseGameReShadeDetectionResult, UseGameReShadeResult } from '@hooks';

import './GameDetailsMetadata.scss';

interface GameDetailsMetadataProps {
  isStorageUsageLoading: boolean;
  modCount: number;
  optiScaler: UseGameOptiScalerResult;
  profileCount: number;
  reShade: UseGameReShadeResult;
  reShadeDetection: UseGameReShadeDetectionResult;
  storageUsedBytes: number | null;
}

const formatStorageUsage = (bytes: number | null, isLoading: boolean) => {
  if (isLoading || bytes === null) {
    return '-';
  }

  if (bytes < 1024) {
    return `${bytes} B`;
  }

  const units = ['KB', 'MB', 'GB', 'TB'];
  let value = bytes / 1024;
  let unitIndex = 0;

  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024;
    unitIndex += 1;
  }

  const formattedValue = value >= 100 ? value.toFixed(0) : value.toFixed(2);
  return `${formattedValue} ${units[unitIndex]}`;
};

const formatOptiScalerStatus = (optiScaler: UseGameOptiScalerResult) => {
  const count = optiScaler.targets.length;
  const countLabel = `${count} managed`;
  switch (optiScaler.aggregateStatus) {
    case 'checking':
      return 'Checking';
    case 'error':
      return 'Error';
    case 'recovery':
      return `${countLabel} · Recovery required`;
    case 'drift':
      return `${countLabel} · Drift detected`;
    case 'update':
      return `${countLabel} · Update available`;
    case 'managed':
      return countLabel;
    case 'unmanaged':
      return 'Detected unmanaged';
    case 'not_detected':
      return 'Not detected';
  }
};

const formatReShadeStatus = (reShadeDetection: UseGameReShadeDetectionResult) => {
  if (reShadeDetection.isLoading) {
    return 'Checking';
  }

  if (reShadeDetection.loadError !== null) {
    return 'Error';
  }

  switch (reShadeDetection.result?.Status) {
    case ReShadeDetectionStatus.ReShadeDetectionStatusInstalled:
      return 'Installed';
    case ReShadeDetectionStatus.ReShadeDetectionStatusNotInstalled:
      return 'Not detected';
    case ReShadeDetectionStatus.ReShadeDetectionStatusUnsupported:
      return 'Unsupported';
    default:
      return '-';
  }
};

const formatManagedReShadeStatus = (reShade: UseGameReShadeResult, fallback: UseGameReShadeDetectionResult) => {
  const count = reShade.targets.length;
  const countLabel = `${count} managed`;
  switch (reShade.aggregateStatus) {
    case 'checking':
      return 'Checking';
    case 'error':
      return formatReShadeStatus(fallback);
    case 'recovery':
      return `${countLabel} · Recovery required`;
    case 'conflict':
      return count > 0 ? `${countLabel} · Conflict` : 'Conflict detected';
    case 'drift':
      return `${countLabel} · Drift detected`;
    case 'update':
      return `${countLabel} · Update available`;
    case 'managed':
      return countLabel;
    case 'unmanaged':
      return 'Detected unmanaged';
    case 'not_detected':
      return formatReShadeStatus(fallback);
  }
};

export const GameDetailsMetadata = ({
  isStorageUsageLoading,
  modCount,
  optiScaler,
  profileCount,
  reShade,
  reShadeDetection,
  storageUsedBytes,
}: GameDetailsMetadataProps) => {
  const metadataItems = [
    { Icon: Package, label: 'Mods installed', value: String(modCount) },
    { Icon: Database, label: 'Storage used', value: formatStorageUsage(storageUsedBytes, isStorageUsageLoading) },
    { Icon: Users, label: 'Profiles', value: String(profileCount) },
    { Icon: Sparkles, label: 'ReShade', value: formatManagedReShadeStatus(reShade, reShadeDetection) },
    { Icon: Gauge, label: 'OptiScaler', value: formatOptiScalerStatus(optiScaler) },
  ];

  return (
    <dl className="game-details-metadata" aria-label="Game metadata">
      {metadataItems.map((item) => (
        <div className="game-details-metadata-item" key={item.label}>
          <dt className="game-details-metadata-label">
            <item.Icon className="game-details-metadata-icon" aria-hidden="true" />
            <span>{item.label}</span>
          </dt>
          <dd className="game-details-metadata-value">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
};
