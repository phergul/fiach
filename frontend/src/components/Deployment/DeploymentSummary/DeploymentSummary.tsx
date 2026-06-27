import type { DeploymentSummary } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatAppliedAtFromDate } from '@utils';

import {
  DEPLOYMENT_SUMMARY_STATUSES,
  resolveDeploymentActionLabel,
  resolveDeploymentSummaryTone,
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
  if (key === 'applied_at') {
    return 'Last applied';
  }

  return resolveDeploymentActionLabel(key);
};

export const DeploymentSummaryBar = ({ summary }: DeploymentSummaryProps) => {
  const items: Array<{ key: string; label: string; value: string | number }> = [];

  if (summary.PlanMode === 'incremental' && summary.AppliedAt !== null) {
    const appliedAtLabel = formatAppliedAtFromDate(summary.AppliedAt);
    if (appliedAtLabel !== 'Applied time unknown') {
      items.push({
        key: 'applied_at',
        label: statusItemLabel('applied_at'),
        value: appliedAtLabel.replace(/^Applied /, ''),
      });
    }
  }

  for (const status of DEPLOYMENT_SUMMARY_STATUSES) {
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
        const tone = resolveDeploymentSummaryTone(item.key) as DeploymentToneChipTone | undefined;
        const isCanApply = item.key === 'can_apply';
        const isAppliedAt = item.key === 'applied_at';

        return (
          <div
            className={
              isCanApply
                ? 'deployment-summary-item deployment-summary-item-can-apply'
                : isAppliedAt
                  ? 'deployment-summary-item deployment-summary-item-applied-at'
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
            ) : isAppliedAt ? (
              <>
                <dt className="deployment-summary-label">{item.label}</dt>
                <dd className="deployment-summary-value">{item.value}</dd>
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
