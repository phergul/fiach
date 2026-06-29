import { describe, expect, it } from 'vitest';

import {
  DeploymentFileInspection,
  InspectionSideMetadata,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import {
  resolveComparePairLabel,
  resolveInspectionKindLabel,
  shouldShowInspectionFallback,
} from './deploymentFileInspectorLabels';

describe('deploymentFileInspectorLabels', () => {
  it('maps inspection kind labels', () => {
    expect(resolveInspectionKindLabel('text_diff')).toBe('Text diff');
    expect(resolveInspectionKindLabel('unknown_kind')).toBe('unknown_kind');
  });

  it('builds compare pair label from side metadata', () => {
    const inspection = new DeploymentFileInspection({
      LeftState: 'current',
      RightState: 'desired',
      Left: new InspectionSideMetadata({ Label: 'Current' }),
      Right: new InspectionSideMetadata({ Label: 'Desired' }),
    });

    expect(resolveComparePairLabel(inspection)).toBe('Current vs Desired');
  });

  it('detects fallback display for binary results', () => {
    const inspection = new DeploymentFileInspection({
      Kind: 'binary_fallback',
      FallbackReason: 'Showing hash and size only.',
    });

    expect(shouldShowInspectionFallback(inspection)).toBe(true);
  });

  it('hides unavailability fallback messages implied by side columns', () => {
    const inspection = new DeploymentFileInspection({
      Kind: 'image_metadata',
      FallbackReason: 'File is not present.',
      Left: new InspectionSideMetadata({
        Label: 'Current',
        Available: false,
        UnavailableReason: 'File is not present.',
      }),
      Right: new InspectionSideMetadata({ Label: 'Desired', Available: true }),
    });

    expect(shouldShowInspectionFallback(inspection)).toBe(false);
  });
});
