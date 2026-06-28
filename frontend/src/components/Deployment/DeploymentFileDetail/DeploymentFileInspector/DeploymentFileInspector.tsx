import type { DeploymentFileInspection } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';

import {
  resolveComparePairLabel,
  resolveInspectionKindLabel,
  shouldShowInspectionFallback,
} from '../deploymentFileInspectorLabels';
import { DeploymentArchiveListing } from './DeploymentArchiveListing/DeploymentArchiveListing';
import { DeploymentBinaryFallback } from './DeploymentBinaryFallback/DeploymentBinaryFallback';
import { DeploymentImageMetadata } from './DeploymentImageMetadata/DeploymentImageMetadata';
import { DeploymentPEMetadata } from './DeploymentPEMetadata/DeploymentPEMetadata';
import { DeploymentTextDiff } from './DeploymentTextDiff/DeploymentTextDiff';

import './DeploymentFileInspector.scss';

interface DeploymentFileInspectorProps {
  inspection: DeploymentFileInspection | null;
  isLoading: boolean;
  loadError: string | null;
  onRetry: () => void;
}

const renderInspectionBody = (inspection: DeploymentFileInspection) => {
  switch (inspection.Kind) {
    case 'text_diff':
      return <DeploymentTextDiff lines={inspection.TextLines} />;
    case 'pe_metadata':
      return (
        <DeploymentPEMetadata
          left={inspection.PEMetadataLeft}
          leftLabel={inspection.Left.Label}
          right={inspection.PEMetadataRight}
          rightLabel={inspection.Right.Label}
        />
      );
    case 'image_metadata':
      return (
        <DeploymentImageMetadata
          left={inspection.ImageMetadataLeft}
          leftLabel={inspection.Left.Label}
          right={inspection.ImageMetadataRight}
          rightLabel={inspection.Right.Label}
        />
      );
    case 'archive_listing':
      return (
        <DeploymentArchiveListing
          leftEntries={inspection.ArchiveEntriesLeft}
          leftLabel={inspection.Left.Label}
          rightEntries={inspection.ArchiveEntriesRight}
          rightLabel={inspection.Right.Label}
        />
      );
    default:
      return <DeploymentBinaryFallback left={inspection.Left} right={inspection.Right} />;
  }
};

export const DeploymentFileInspector = ({
  inspection,
  isLoading,
  loadError,
  onRetry,
}: DeploymentFileInspectorProps) => {
  if (isLoading) {
    return <StateBlock message="Loading file comparison…" title="Loading" />;
  }

  if (loadError !== null) {
    return (
      <div className="deployment-file-inspector-error">
        <StateBlock message={loadError} title="Could not load file comparison" />
        <button className="deployment-file-inspector-retry" onClick={onRetry} type="button">
          Retry
        </button>
      </div>
    );
  }

  if (inspection === null) {
    return <StateBlock message="File comparison is not available for this path." title="No comparison" />;
  }

  const compareLabel = resolveComparePairLabel(inspection);

  return (
    <div className="deployment-file-inspector">
      <div className="deployment-file-inspector-summary">
        <p className="deployment-file-inspector-kind">{resolveInspectionKindLabel(inspection.Kind)}</p>
        {compareLabel !== '' && (
          <p className="deployment-file-inspector-pair">{compareLabel}</p>
        )}
      </div>

      {shouldShowInspectionFallback(inspection) && inspection.FallbackReason !== '' && (
        <p className="deployment-file-inspector-fallback">{inspection.FallbackReason}</p>
      )}

      {inspection.LimitReached && inspection.LimitReason !== '' && (
        <p className="deployment-file-inspector-limit">{inspection.LimitReason}</p>
      )}

      {renderInspectionBody(inspection)}
    </div>
  );
};
