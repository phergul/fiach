import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';

import { DropdownMenu } from './DropdownMenu';

describe('DropdownMenu', () => {
  it('keeps grouped actions in a nested submenu', () => {
    const onSelect = vi.fn();
    render(
      <DropdownMenu
        ariaLabel="Game actions"
        isOpen
        items={[{
          children: [
            { label: 'OptiScaler', onSelect },
            { label: 'Install ReShade', onSelect },
          ],
          label: 'Manage graphics tools',
        }]}
      />,
    );

    expect(screen.queryByRole('menuitem', { name: 'OptiScaler' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Manage graphics tools' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'OptiScaler' }));
    expect(onSelect).toHaveBeenCalledOnce();
  });
});
