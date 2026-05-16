import type { ComponentType } from 'react';

import type { LucideProps } from 'lucide-react';

import './DropdownMenu.scss';

export interface DropdownMenuItem {
  disabled?: boolean;
  icon?: ComponentType<LucideProps>;
  label: string;
  onSelect: () => void;
}

interface DropdownMenuProps {
  align?: 'left' | 'right';
  ariaLabel: string;
  isOpen: boolean;
  items: DropdownMenuItem[];
}

export const DropdownMenu = ({
  align = 'right',
  ariaLabel,
  isOpen,
  items,
}: DropdownMenuProps) => {
  if (!isOpen) {
    return null;
  }

  return (
    <div className={`dropdown-menu dropdown-menu-${align}`} role="menu" aria-label={ariaLabel}>
      {items.map((item) => {
        const Icon = item.icon;

        return (
          <button
            className="dropdown-menu-item"
            disabled={item.disabled}
            key={item.label}
            onClick={item.onSelect}
            role="menuitem"
            type="button"
          >
            {Icon !== undefined && <Icon className="dropdown-menu-icon" aria-hidden="true" />}
            {item.label}
          </button>
        );
      })}
    </div>
  );
};
