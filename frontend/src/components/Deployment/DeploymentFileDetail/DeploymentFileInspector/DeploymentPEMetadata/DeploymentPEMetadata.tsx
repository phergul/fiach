import type { PEMetadata } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { formatDeploymentBytes, truncateDeploymentHash } from '@utils';

import {
  DeploymentCompareColumn,
  DeploymentCompareColumnPlaceholder,
  DeploymentCompareGrid,
} from '../DeploymentCompareColumn/DeploymentCompareColumn';

interface DeploymentPEMetadataProps {
  left: PEMetadata | null;
  leftLabel: string;
  right: PEMetadata | null;
  rightLabel: string;
}

const MetadataBody = ({ metadata }: { metadata: PEMetadata }) => (
  <dl className="deployment-compare-column-details">
    <div className="deployment-compare-column-detail">
      <dt>Machine</dt>
      <dd>{metadata.Machine}</dd>
    </div>
    <div className="deployment-compare-column-detail">
      <dt>Sections</dt>
      <dd>{metadata.SectionCount}</dd>
    </div>
    <div className="deployment-compare-column-detail">
      <dt>Characteristics</dt>
      <dd>{metadata.Characteristics}</dd>
    </div>
    <div className="deployment-compare-column-detail">
      <dt>Type</dt>
      <dd>{metadata.IsDLL ? 'DLL' : metadata.IsEXE ? 'EXE' : 'PE'}</dd>
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

export const DeploymentPEMetadata = ({
  left,
  leftLabel,
  right,
  rightLabel,
}: DeploymentPEMetadataProps) => (
  <DeploymentCompareGrid aria-label="PE metadata comparison">
    <DeploymentCompareColumn isEmpty={left === null} label={leftLabel}>
      {left === null ? <DeploymentCompareColumnPlaceholder /> : <MetadataBody metadata={left} />}
    </DeploymentCompareColumn>
    <DeploymentCompareColumn isDesired isEmpty={right === null} label={rightLabel}>
      {right === null ? <DeploymentCompareColumnPlaceholder /> : <MetadataBody metadata={right} />}
    </DeploymentCompareColumn>
  </DeploymentCompareGrid>
);
