import type { DeploymentFileInspection } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

const inspectionKindLabels: Record<string, string> = {
  text_diff: 'Text diff',
  pe_metadata: 'PE metadata',
  image_metadata: 'Image metadata',
  archive_listing: 'Archive listing',
  binary_fallback: 'Binary metadata',
};

export const resolveInspectionKindLabel = (kind: string): string =>
  inspectionKindLabels[kind] ?? kind;

export const resolveComparePairLabel = (inspection: DeploymentFileInspection): string => {
  const leftLabel = inspection.Left?.Label ?? inspection.LeftState;
  const rightLabel = inspection.Right?.Label ?? inspection.RightState;

  if (leftLabel === '' && rightLabel === '') {
    return '';
  }

  return `${leftLabel} vs ${rightLabel}`;
};

export const shouldShowInspectionFallback = (inspection: DeploymentFileInspection): boolean => {
  const reason = inspection.FallbackReason.trim();
  if (reason === '') {
    return false;
  }

  if (
    reason === inspection.Left?.UnavailableReason ||
    reason === inspection.Right?.UnavailableReason
  ) {
    return false;
  }

  if (reason === 'File is not present.' || reason === 'Neither side is available for inspection.') {
    return false;
  }

  return true;
};
