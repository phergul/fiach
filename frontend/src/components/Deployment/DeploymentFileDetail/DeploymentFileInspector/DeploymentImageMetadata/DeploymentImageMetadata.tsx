import type { ImageMetadata } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentBytes, truncateDeploymentHash } from '@utils';

import {
  DeploymentCompareColumn,
  DeploymentCompareColumnPlaceholder,
  DeploymentCompareGrid,
} from '../DeploymentCompareColumn/DeploymentCompareColumn';

interface DeploymentImageMetadataProps {
  left: ImageMetadata | null;
  leftLabel: string;
  right: ImageMetadata | null;
  rightLabel: string;
}

const MetadataBody = ({ metadata }: { metadata: ImageMetadata }) => (
  <dl className="deployment-compare-column-details">
    <div className="deployment-compare-column-detail">
      <dt>Format</dt>
      <dd>{metadata.Format.toUpperCase()}</dd>
    </div>
    <div className="deployment-compare-column-detail">
      <dt>Dimensions</dt>
      <dd>
        {metadata.Width} × {metadata.Height}
      </dd>
    </div>
    <div className="deployment-compare-column-detail">
      <dt>Size</dt>
      <dd>{formatDeploymentBytes(metadata.SizeBytes)}</dd>
    </div>
    {metadata.SHA256 !== '' && (
      <div className="deployment-compare-column-detail">
        <dt>SHA256</dt>
        <dd className="deployment-compare-column-hash" title={metadata.SHA256}>
          {truncateDeploymentHash(metadata.SHA256)}
        </dd>
      </div>
    )}
  </dl>
);

export const DeploymentImageMetadata = ({
  left,
  leftLabel,
  right,
  rightLabel,
}: DeploymentImageMetadataProps) => (
  <DeploymentCompareGrid aria-label="Image metadata comparison">
    <DeploymentCompareColumn isEmpty={left === null} label={leftLabel}>
      {left === null ? <DeploymentCompareColumnPlaceholder /> : <MetadataBody metadata={left} />}
    </DeploymentCompareColumn>
    <DeploymentCompareColumn isDesired isEmpty={right === null} label={rightLabel}>
      {right === null ? <DeploymentCompareColumnPlaceholder /> : <MetadataBody metadata={right} />}
    </DeploymentCompareColumn>
  </DeploymentCompareGrid>
);
