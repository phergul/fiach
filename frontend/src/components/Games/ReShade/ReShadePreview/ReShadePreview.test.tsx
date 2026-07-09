import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import {
  Ownership,
  PathRole,
  type Preview as ReShadePreviewModel,
} from '@bindings/github.com/phergul/fiach/internal/reshade/models';

import { ReShadePreview } from './ReShadePreview';

const previewBase = (): ReShadePreviewModel =>
  ({
    canApply: true,
    conflicts: [],
    desiredTarget: null,
    drift: [],
    operations: [],
    pathImpacts: [],
    previewHash: 'preview-hash',
    request: {
      targetRelativePath: 'System',
    },
    userContentDrift: [],
    warnings: [],
  }) as unknown as ReShadePreviewModel;

describe('ReShadePreview', () => {
  it('shows missing replacement impacts as target-relative additions', () => {
    render(
      <ReShadePreview
        chainTarget={null}
        preview={{
          ...previewBase(),
          pathImpacts: [
            {
              action: 'replace',
              blocking: false,
              exists: false,
              ownership: Ownership.OwnershipManaged,
              path: 'System\\d3d9.dll',
              preservationOnly: false,
              role: PathRole.PathRoleRuntime,
            },
          ],
        }}
      />,
    );

    expect(screen.getByText('add: d3d9.dll')).toBeInTheDocument();
    expect(screen.queryByText('replace: System\\d3d9.dll')).not.toBeInTheDocument();
  });

  it('hides preserve-only configuration and user content impacts', () => {
    render(
      <ReShadePreview
        chainTarget={null}
        preview={{
          ...previewBase(),
          pathImpacts: [
            {
              action: 'preserve',
              blocking: false,
              exists: true,
              ownership: Ownership.OwnershipUser,
              path: 'ReShade.ini',
              preservationOnly: true,
              role: PathRole.PathRoleConfiguration,
            },
            {
              action: 'preserve',
              blocking: false,
              exists: true,
              ownership: Ownership.OwnershipUser,
              path: 'System\\ReShade.ini',
              preservationOnly: true,
              role: PathRole.PathRoleConfiguration,
            },
          ],
        }}
      />,
    );

    expect(screen.queryByText('Configuration and presets')).not.toBeInTheDocument();
    expect(screen.queryByText(/preserve/i)).not.toBeInTheDocument();
  });

  it('normalizes configuration impacts to the selected target path', () => {
    render(
      <ReShadePreview
        chainTarget={null}
        preview={{
          ...previewBase(),
          pathImpacts: [
            {
              action: 'update search paths',
              blocking: false,
              exists: true,
              ownership: Ownership.OwnershipUser,
              path: 'System\\ReShade.ini',
              preservationOnly: false,
              role: PathRole.PathRoleConfiguration,
            },
          ],
        }}
      />,
    );

    expect(screen.getByText('update: ReShade.ini')).toBeInTheDocument();
    expect(screen.queryByText('System\\ReShade.ini')).not.toBeInTheDocument();
  });

  it('shows copy operations as target files instead of source-to-target pairs', () => {
    render(
      <ReShadePreview
        chainTarget={null}
        preview={{
          ...previewBase(),
          operations: [
            {
              sourcePath: 'C:/Fiach/staging/d3d9.dll',
              targetPath: 'C:/Games/Witcher/System/d3d9.dll',
              type: 'copy',
            },
          ],
        }}
      />,
    );

    expect(screen.getByText('Files to add or update')).toBeInTheDocument();
    expect(screen.getByText(/d3d9\.dll/)).toBeInTheDocument();
    expect(screen.queryByText(/d3d9\.dll -> d3d9\.dll/)).not.toBeInTheDocument();
  });
});
