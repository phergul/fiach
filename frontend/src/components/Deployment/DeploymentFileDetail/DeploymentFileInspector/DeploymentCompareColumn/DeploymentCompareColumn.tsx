import type { ReactNode } from 'react';

import './DeploymentCompareColumn.scss';

interface DeploymentCompareGridProps {
  'aria-label'?: string;
  children: ReactNode;
}

export const DeploymentCompareGrid = ({ 'aria-label': ariaLabel, children }: DeploymentCompareGridProps) => (
  <div aria-label={ariaLabel} className="deployment-compare-grid" role={ariaLabel !== undefined ? 'group' : undefined}>
    {children}
  </div>
);

interface DeploymentCompareColumnProps {
  children: ReactNode;
  isDesired?: boolean;
  isEmpty?: boolean;
  label: string;
}

export const DeploymentCompareColumn = ({
  children,
  isDesired = false,
  isEmpty = false,
  label,
}: DeploymentCompareColumnProps) => (
  <article
    className={[
      'deployment-compare-column',
      isDesired ? 'deployment-compare-column-desired' : '',
    ]
      .filter(Boolean)
      .join(' ')}
  >
    <header className="deployment-compare-column-header">{label}</header>
    <div
      className={
        isEmpty
          ? 'deployment-compare-column-body deployment-compare-column-body-empty'
          : 'deployment-compare-column-body deployment-compare-column-body-populated'
      }
    >
      {children}
    </div>
  </article>
);

export const DeploymentCompareColumnPlaceholder = () => (
  <p className="deployment-compare-column-placeholder">Not available</p>
);
