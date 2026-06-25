import { RefreshCw } from 'lucide-react';

import type { ReShadeInstallerStatus } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ReShadePageHeader.scss';

interface ReShadePageHeaderProps {
  installerStatus: ReShadeInstallerStatus | null;
  isLoading: boolean;
  onRefresh: () => void;
}

const formatRuntimeVersion = (version: string | null | undefined) => {
  const trimmed = version?.trim() ?? '';
  if (trimmed === '') {
    return '';
  }
  return trimmed.toLowerCase().startsWith('v') ? trimmed : `v${trimmed}`;
};

const latestRemoteReleaseLabel = (installerStatus: ReShadeInstallerStatus | null) => {
  const version = formatRuntimeVersion(installerStatus?.standard?.version);
  if (version === '') {
    return 'Latest release unavailable';
  }
  return version;
};

export const ReShadePageHeader = ({
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
      <button disabled={isLoading} onClick={onRefresh} type="button">
        <RefreshCw aria-hidden="true" />
        Refresh
      </button>
    </div>
  </header>
);
