import type { DeploymentSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import {
  DEPLOYMENT_FILE_STATUSES,
  deploymentStatusLabel,
  deploymentSummaryTone,
  type DeploymentToneChipTone,
} from '../deploymentLabels';
import { DeploymentToneChip } from '../DeploymentToneChip/DeploymentToneChip';

import './DeploymentSummary.scss';

interface DeploymentSummaryProps {
  summary: DeploymentSummary;
}

const statusItemLabel = (key: string) => {
  if (key === 'blocking') {
    return 'Blocking';
  }
  if (key === 'warnings') {
    return 'Warnings';
  }
  if (key === 'can_apply') {
    return 'Can apply';
  }

  return deploymentStatusLabel[key] ?? key;
};

export const DeploymentSummaryBar = ({ summary }: DeploymentSummaryProps) => {
  const items: Array<{ key: string; label: string; value: string | number }> = [];

  for (const status of DEPLOYMENT_FILE_STATUSES) {
    const count = summary.StatusCounts[status] ?? 0;
    if (count > 0) {
      items.push({
        key: status,
        label: statusItemLabel(status),
        value: count,
      });
    }
  }

  if (summary.BlockingCount > 0) {
    items.push({
      key: 'blocking',
      label: statusItemLabel('blocking'),
      value: summary.BlockingCount,
    });
  }

  if (summary.WarningCount > 0) {
    items.push({
      key: 'warnings',
      label: statusItemLabel('warnings'),
      value: summary.WarningCount,
    });
  }

  items.push({
    key: 'can_apply',
    label: statusItemLabel('can_apply'),
    value: summary.CanApply ? 'Yes' : 'No',
  });

  return (
    <dl className="deployment-summary" aria-label="Deployment review summary">
      {items.map((item) => {
        const tone = deploymentSummaryTone[item.key] as DeploymentToneChipTone | undefined;
        const isCanApply = item.key === 'can_apply';

        return (
          <div
            className={
              isCanApply
                ? 'deployment-summary-item deployment-summary-item-can-apply'
                : 'deployment-summary-item'
            }
            key={item.key}
          >
            {isCanApply ? (
              <>
                <dt className="deployment-summary-label">{item.label}</dt>
                <dd
                  className={`deployment-summary-value deployment-summary-value-${summary.CanApply ? 'ready' : 'blocked'}`}
                >
                  {item.value}
                </dd>
              </>
            ) : (
              <DeploymentToneChip label={`${item.label} ${item.value}`} tone={tone ?? 'default'} />
            )}
          </div>
        );
      })}
    </dl>
  );
};
