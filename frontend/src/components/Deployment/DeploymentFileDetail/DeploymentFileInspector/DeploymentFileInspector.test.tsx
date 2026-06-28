import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import {
  DeploymentFileInspection,
  InspectionSideMetadata,
  TextDiffLine,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { DeploymentFileInspector } from './DeploymentFileInspector';

describe('DeploymentFileInspector', () => {
  it('renders loading state', () => {
    render(
      <DeploymentFileInspector
        inspection={null}
        isLoading
        loadError={null}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByText('Loading file comparison…')).toBeInTheDocument();
  });

  it('renders text diff lines', () => {
    render(
      <DeploymentFileInspector
        inspection={
          new DeploymentFileInspection({
            Kind: 'text_diff',
            Left: new InspectionSideMetadata({ Label: 'Current', Available: true }),
            Right: new InspectionSideMetadata({ Label: 'Desired', Available: true }),
            TextLines: [
              new TextDiffLine({ Kind: 'delete', Line: 'enabled=0', LineNo: 1 }),
              new TextDiffLine({ Kind: 'insert', Line: 'enabled=1', LineNo: 1 }),
            ],
          })
        }
        isLoading={false}
        loadError={null}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByText('Text diff')).toBeInTheDocument();
    expect(screen.getByText('enabled=0')).toBeInTheDocument();
    expect(screen.getByText('enabled=1')).toBeInTheDocument();
  });

  it('renders binary fallback metadata', () => {
    render(
      <DeploymentFileInspector
        inspection={
          new DeploymentFileInspection({
            Kind: 'binary_fallback',
            FallbackReason: 'Showing hash and size only.',
            Left: new InspectionSideMetadata({
              Label: 'Current',
              Available: true,
              SHA256: 'abc123',
              SizeBytes: 12,
            }),
            Right: new InspectionSideMetadata({
              Label: 'Desired',
              Available: true,
              SHA256: 'def456',
              SizeBytes: 14,
            }),
          })
        }
        isLoading={false}
        loadError={null}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByText('Showing hash and size only.')).toBeInTheDocument();
    expect(screen.getByText('12 B')).toBeInTheDocument();
    expect(screen.getByText('14 B')).toBeInTheDocument();
  });

  it('hides implied unavailability fallback for image metadata', () => {
    render(
      <DeploymentFileInspector
        inspection={
          new DeploymentFileInspection({
            Kind: 'image_metadata',
            FallbackReason: 'File is not present.',
            Left: new InspectionSideMetadata({
              Label: 'Current',
              Available: false,
              UnavailableReason: 'File is not present.',
            }),
            Right: new InspectionSideMetadata({ Label: 'Desired', Available: true }),
            ImageMetadataRight: {
              Format: 'jpeg',
              Width: 3840,
              Height: 2160,
              SHA256: 'abc',
              SizeBytes: 1700000,
            },
          })
        }
        isLoading={false}
        loadError={null}
        onRetry={() => undefined}
      />,
    );

    expect(screen.getByText('Image metadata')).toBeInTheDocument();
    expect(screen.getByText('Current vs Desired')).toBeInTheDocument();
    expect(screen.queryByText('File is not present.')).not.toBeInTheDocument();
    expect(screen.getByText('Not available')).toBeInTheDocument();
  });
});
