import { useEffect, useState } from 'react';
import type { ComponentType } from 'react';

import { ChevronRight, type LucideProps } from 'lucide-react';

import './DropdownMenu.scss';

export interface DropdownMenuItem {
  checked?: boolean;
  children?: DropdownMenuItem[];
  disabled?: boolean;
  icon?: ComponentType<LucideProps>;
  label: string;
  onSelect?: () => void;
  type?: 'action' | 'checkbox';
}

interface DropdownMenuProps {
  align?: 'left' | 'right';
  ariaLabel: string;
  isOpen: boolean;
  items: DropdownMenuItem[];
}

export const DropdownMenu = ({ align = 'right', ariaLabel, isOpen, items }: DropdownMenuProps) => {
  const [openSubmenuLabel, setOpenSubmenuLabel] = useState<string | null>(null);

  useEffect(() => {
    if (!isOpen) {
      setOpenSubmenuLabel(null);
    }
  }, [isOpen]);

  if (!isOpen) {
    return null;
  }

  return (
    <div className={`dropdown-menu dropdown-menu-${align}`} role="menu" aria-label={ariaLabel}>
      {items.map((item) => {
        if (item.type === 'checkbox') {
          return (
            <label className="dropdown-menu-checkbox-option" key={item.label}>
              <input
                checked={item.checked ?? false}
                disabled={item.disabled}
                onChange={() => item.onSelect?.()}
                type="checkbox"
              />
              <span className="dropdown-menu-checkbox-control" aria-hidden="true" />
              <span className="dropdown-menu-item-label">{item.label}</span>
            </label>
          );
        }

        const Icon = item.icon;
        const hasSubmenu = item.children !== undefined && item.children.length > 0;
        const isSubmenuOpen = hasSubmenu && openSubmenuLabel === item.label;

        return (
          <div className="dropdown-menu-entry" key={item.label}>
            <button
              aria-expanded={hasSubmenu ? isSubmenuOpen : undefined}
              aria-haspopup={hasSubmenu ? 'menu' : undefined}
              className="dropdown-menu-item"
              disabled={item.disabled}
              onClick={() => {
                if (hasSubmenu) {
                  setOpenSubmenuLabel(isSubmenuOpen ? null : item.label);
                  return;
                }
                item.onSelect?.();
              }}
              role="menuitem"
              type="button"
            >
              {Icon !== undefined && <Icon className="dropdown-menu-icon" aria-hidden="true" />}
              <span className="dropdown-menu-item-label">{item.label}</span>
              {hasSubmenu && (
                <ChevronRight className="dropdown-menu-submenu-icon" aria-hidden="true" />
              )}
            </button>

            {isSubmenuOpen && (
              <div
                className="dropdown-menu dropdown-menu-submenu"
                role="menu"
                aria-label={item.label}
              >
                {item.children?.map((child) => {
                  const ChildIcon = child.icon;
                  return (
                    <button
                      className="dropdown-menu-item"
                      disabled={child.disabled}
                      key={child.label}
                      onClick={child.onSelect}
                      role="menuitem"
                      type="button"
                    >
                      {ChildIcon !== undefined && (
                        <ChildIcon className="dropdown-menu-icon" aria-hidden="true" />
                      )}
                      <span className="dropdown-menu-item-label">{child.label}</span>
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
