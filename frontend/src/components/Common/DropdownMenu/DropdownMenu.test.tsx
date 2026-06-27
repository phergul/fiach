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
        items={[
          {
            children: [
              { label: 'OptiScaler', onSelect },
              { label: 'Manage ReShade', onSelect },
            ],
            label: 'Manage graphics tools',
          },
        ]}
      />,
    );

    expect(screen.queryByRole('menuitem', { name: 'OptiScaler' })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('menuitem', { name: 'Manage graphics tools' }));
    fireEvent.click(screen.getByRole('menuitem', { name: 'OptiScaler' }));
    expect(onSelect).toHaveBeenCalledOnce();
  });

  it('renders checkbox options', () => {
    const onToggle = vi.fn();
    render(
      <DropdownMenu
        ariaLabel="Status filter"
        isOpen
        items={[
          {
            checked: true,
            label: 'Added',
            onSelect: onToggle,
            type: 'checkbox',
          },
        ]}
      />,
    );

    expect(screen.getByRole('checkbox', { name: 'Added' })).toBeChecked();
    fireEvent.click(screen.getByRole('checkbox', { name: 'Added' }));
    expect(onToggle).toHaveBeenCalledOnce();
  });
});
