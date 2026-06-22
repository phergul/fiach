import { RefreshCw } from 'lucide-react';

import type {
  ManagedReShadeInstallerStatus,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import type { ReShadeAggregateStatus } from '@hooks';

import './ReShadePageHeader.scss';

interface ReShadePageHeaderProps {
  aggregateStatus: ReShadeAggregateStatus;
  installerStatus: ManagedReShadeInstallerStatus | null;
  isLoading: boolean;
  onRefresh: () => void;
}

const statusLabel: Record<ReShadeAggregateStatus, string> = {
  checking: 'Checking',
  error: 'Error',
  recovery: 'Recovery required',
  conflict: 'Conflict',
  drift: 'Drift detected',
  update: 'Update available',
  managed: 'Managed',
  unmanaged: 'Detected unmanaged',
  not_detected: 'Not detected',
};

const formatRuntimeVersion = (version: string | null | undefined) => {
  const trimmed = version?.trim() ?? '';
  if (trimmed === '') {
    return '';
  }
  return trimmed.toLowerCase().startsWith('v') ? trimmed : `v${trimmed}`;
};

const latestRemoteReleaseLabel = (installerStatus: ManagedReShadeInstallerStatus | null) => {
  const standard = installerStatus?.standard;
  const addon = installerStatus?.addon;
  console.log('Installer status:', installerStatus);
  const versions = [
    standard !== undefined && standard.error === undefined ? formatRuntimeVersion(standard.version) : '',
    addon !== undefined && addon.error === undefined ? formatRuntimeVersion(addon.version) : '',
  ].filter((version) => version !== '');
  const uniqueVersions = [...new Set(versions)];
  if (uniqueVersions.length === 0) {
    return 'Latest release unavailable';
  }
  return `${uniqueVersions.join(', ')}`;
};

export const ReShadePageHeader = ({
  aggregateStatus,
  installerStatus,
  isLoading,
  onRefresh,
}: ReShadePageHeaderProps) => (
  <header className="reshade-page-header">
    <div className="reshade-page-header-title">
      <h2>ReShade</h2>
      <p>{latestRemoteReleaseLabel(installerStatus)}</p>
    </div>
    <div className="reshade-page-header-actions">
      <span className={`reshade-page-header-status reshade-page-header-status-${aggregateStatus}`}>
        {statusLabel[aggregateStatus]}
      </span>
      <button disabled={isLoading} onClick={onRefresh} type="button">
        <RefreshCw aria-hidden="true" />
        Refresh
      </button>
    </div>
  </header>
);
