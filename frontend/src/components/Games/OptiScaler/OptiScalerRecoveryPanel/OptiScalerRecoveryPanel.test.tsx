import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { OptiScalerRecoveryPanel } from './OptiScalerRecoveryPanel';

describe('OptiScalerRecoveryPanel', () => {
  it('shows recovery context and exposes rollback as the only operation', () => {
    const onRollback = vi.fn();
    render(
      <OptiScalerRecoveryPanel
        isRollingBack={false}
        onRollback={onRollback}
        recovery={{
          action: 'update',
          error: 'Rollback verification failed',
          gameId: 7,
          journalId: 'journal-1',
          required: true,
          targetPath: 'Game/Binaries/Win64',
        } as never}
      />,
    );

    expect(screen.getByText('Recovery required')).toBeInTheDocument();
    expect(screen.getByText('Rollback verification failed')).toBeInTheDocument();
    expect(screen.getByText('Game/Binaries/Win64')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: 'Roll back operation' }));
    expect(onRollback).toHaveBeenCalledOnce();
  });
});
