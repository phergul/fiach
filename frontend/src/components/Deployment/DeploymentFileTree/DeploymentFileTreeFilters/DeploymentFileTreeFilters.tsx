import { Search } from 'lucide-react';

import {
  DEPLOYMENT_FILTER_STATUSES,
  DEPLOYMENT_RISK_LEVELS,
  deploymentRiskLabel,
  resolveDeploymentActionLabel,
} from '../../deploymentLabels';

import { DeploymentTreeFilterDropdown } from './DeploymentTreeFilterDropdown';

import './DeploymentFileTreeFilters.scss';

interface DeploymentFileTreeFiltersProps {
  filters: {
    risks: string[];
    searchQuery: string;
    statuses: string[];
  };
  isScanning: boolean;
  onFiltersChange: (filters: {
    risks: string[];
    searchQuery: string;
    statuses: string[];
  }) => void;
  scanCapReached: boolean;
}

export const DeploymentFileTreeFilters = ({
  filters,
  isScanning,
  onFiltersChange,
  scanCapReached,
}: DeploymentFileTreeFiltersProps) => {
  return (
    <section className="deployment-file-tree-filters" aria-label="Deployment tree filters">
      <div className="deployment-file-tree-filters-row">
        <label className="deployment-file-tree-filters-search">
          <Search className="deployment-file-tree-filters-search-icon" aria-hidden="true" />
          <input
            className="deployment-file-tree-filters-input"
            onChange={(event) =>
              onFiltersChange({
                ...filters,
                searchQuery: event.target.value,
              })
            }
            placeholder="Filter by path or name"
            type="search"
            value={filters.searchQuery}
          />
        </label>

        <div className="deployment-file-tree-filters-dropdowns">
          <DeploymentTreeFilterDropdown
            label="Status"
            options={DEPLOYMENT_FILTER_STATUSES.map((status) => ({
              label: resolveDeploymentActionLabel(status),
              value: status,
            }))}
            selectedValues={filters.statuses}
            onChange={(statuses) =>
              onFiltersChange({
                ...filters,
                statuses,
              })
            }
          />
          <DeploymentTreeFilterDropdown
            label="Risk"
            options={DEPLOYMENT_RISK_LEVELS.map((risk) => ({
              label: deploymentRiskLabel[risk],
              value: risk,
            }))}
            selectedValues={filters.risks}
            onChange={(risks) =>
              onFiltersChange({
                ...filters,
                risks,
              })
            }
          />
        </div>
      </div>

      {isScanning && <p className="deployment-file-tree-filters-status">Scanning files…</p>}
      {scanCapReached && (
        <p className="deployment-file-tree-filters-status deployment-file-tree-filters-status-warning">
          Large deployment — refine search to narrow results.
        </p>
      )}
    </section>
  );
};
