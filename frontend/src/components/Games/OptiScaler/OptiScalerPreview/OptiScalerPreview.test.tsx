import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

import type { OptiScalerPreview as OptiScalerPreviewModel } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { OptiScalerPreview } from './OptiScalerPreview';

describe('OptiScalerPreview', () => {
  it('shows backup destinations, conflicts, drift, and retained uninstall state', () => {
    const preview = {
      canApply: false,
      configurationChanges: [],
      conflicts: ['Unknown proxy ownership'],
      drift: [{ actualHash: 'b', expectedHash: 'a', missing: false, relativePath: 'dxgi.dll' }],
      operations: [{
        backupPath: 'C:/Fiach/backups/dxgi.bak',
        targetPath: 'C:/Game/dxgi.dll',
        type: 'copy',
      }],
      request: { action: 'uninstall' },
      warnings: [],
    } as unknown as OptiScalerPreviewModel;

    render(<OptiScalerPreview preview={preview} />);

    expect(screen.getByText('Unknown proxy ownership')).toBeInTheDocument();
    expect(screen.getByText('dxgi.dll has changed')).toBeInTheDocument();
    expect(screen.getByText('Backup: C:/Fiach/backups/dxgi.bak')).toBeInTheDocument();
    expect(screen.getByText('Retained after uninstall')).toBeInTheDocument();
  });
});
