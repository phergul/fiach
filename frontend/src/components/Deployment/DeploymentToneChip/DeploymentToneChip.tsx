import type { DeploymentToneChipTone } from '../deploymentLabels';

import './DeploymentToneChip.scss';

interface DeploymentToneChipProps {
  label: string;
  tone?: DeploymentToneChipTone;
}

export const DeploymentToneChip = ({ label, tone = 'default' }: DeploymentToneChipProps) => {
  return <span className={`deployment-tone-chip deployment-tone-chip-${tone}`}>{label}</span>;
};
