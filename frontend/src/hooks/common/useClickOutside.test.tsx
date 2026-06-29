import { fireEvent, render, screen } from '@testing-library/react';
import { useRef, useState } from 'react';
import { describe, expect, it } from 'vitest';

import { useClickOutside } from './useClickOutside';

const ClickOutsideFixture = () => {
  const [isOpen, setIsOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  useClickOutside(containerRef, () => setIsOpen(false), isOpen);

  return (
    <div>
      <div ref={containerRef}>
        <button onClick={() => setIsOpen(true)} type="button">
          Open
        </button>
        {isOpen && <p>Popover</p>}
      </div>
      <button type="button">Outside</button>
    </div>
  );
};

describe('useClickOutside', () => {
  it('closes when clicking outside the container', () => {
    render(<ClickOutsideFixture />);

    fireEvent.click(screen.getByRole('button', { name: 'Open' }));
    expect(screen.getByText('Popover')).toBeInTheDocument();

    fireEvent.pointerDown(screen.getByRole('button', { name: 'Outside' }));
    expect(screen.queryByText('Popover')).not.toBeInTheDocument();
  });

  it('does not close when clicking inside the container', () => {
    render(<ClickOutsideFixture />);

    fireEvent.click(screen.getByRole('button', { name: 'Open' }));
    fireEvent.pointerDown(screen.getByText('Popover'));
    expect(screen.getByText('Popover')).toBeInTheDocument();
  });
});
