import { RefreshCw } from 'lucide-react';

import type { OptiScalerRelease } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './OptiScalerPageHeader.scss';

interface OptiScalerPageHeaderProps {
  isLoading: boolean;
  onRefresh: () => void;
  release: OptiScalerRelease | null;
}

export const OptiScalerPageHeader = ({
  isLoading,
  onRefresh,
  release,
}: OptiScalerPageHeaderProps) => {
  const version = release?.version || release?.tag;

  return (
    <header className="optiscaler-page-header">
      <div>
        <div className="optiscaler-page-header-title">
          <h2>OptiScaler</h2>
          {version && <span>{version}</span>}
        </div>
        <p>Manage each executable directory independently and review every file change before applying.</p>
      </div>
      <button disabled={isLoading} onClick={onRefresh} type="button">
        <RefreshCw aria-hidden="true" />
        Refresh
      </button>
    </header>
  );
};
