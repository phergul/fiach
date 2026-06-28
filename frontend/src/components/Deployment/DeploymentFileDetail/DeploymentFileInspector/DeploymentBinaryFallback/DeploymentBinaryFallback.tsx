import type { InspectionSideMetadata } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentBytes, truncateDeploymentHash } from '@utils';

import {
  DeploymentCompareColumn,
  DeploymentCompareColumnPlaceholder,
  DeploymentCompareGrid,
} from '../DeploymentCompareColumn/DeploymentCompareColumn';

interface DeploymentBinaryFallbackProps {
  left: InspectionSideMetadata;
  right: InspectionSideMetadata;
}

const SideBody = ({ side }: { side: InspectionSideMetadata }) => {
  if (!side.Available) {
    return <DeploymentCompareColumnPlaceholder />;
  }

  return (
    <dl className="deployment-compare-column-details">
      <div className="deployment-compare-column-detail">
        <dt>Size</dt>
        <dd>{formatDeploymentBytes(side.SizeBytes)}</dd>
      </div>
      {side.SHA256 !== '' && (
        <div className="deployment-compare-column-detail">
          <dt>SHA256</dt>
          <dd className="deployment-compare-column-hash" title={side.SHA256}>
            {truncateDeploymentHash(side.SHA256)}
          </dd>
        </div>
      )}
    </dl>
  );
};

export const DeploymentBinaryFallback = ({ left, right }: DeploymentBinaryFallbackProps) => (
  <DeploymentCompareGrid aria-label="Binary metadata comparison">
    <DeploymentCompareColumn isEmpty={!left.Available} label={left.Label}>
      <SideBody side={left} />
    </DeploymentCompareColumn>
    <DeploymentCompareColumn isDesired isEmpty={!right.Available} label={right.Label}>
      <SideBody side={right} />
    </DeploymentCompareColumn>
  </DeploymentCompareGrid>
);
