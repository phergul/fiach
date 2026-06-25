import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';

import { Navigation } from '@components/Navigation/Navigation';

import './Sidebar.scss';

interface SidebarProps {
  isPinned: boolean;
  onPinnedChange: (isPinned: boolean) => void;
}

export const Sidebar = ({ isPinned, onPinnedChange }: SidebarProps) => {
  const pinLabel = isPinned ? 'Unpin sidebar' : 'Pin sidebar';

  return (
    <aside
      className={isPinned ? 'sidebar sidebar-pinned' : 'sidebar'}
      aria-label="Primary navigation"
    >
      <div className="sidebar-surface">
        <Navigation />
        <div className="sidebar-pin-section">
          <span className="sidebar-separator" aria-hidden="true" />
          <button
            aria-label={pinLabel}
            className="sidebar-pin-button"
            onClick={(event) => {
              onPinnedChange(!isPinned);

              if (event.detail > 0) {
                event.currentTarget.blur();
              }
            }}
            title={pinLabel}
            type="button"
          >
            {isPinned ? (
              <PanelLeftClose className="sidebar-pin-button-icon" aria-hidden="true" />
            ) : (
              <PanelLeftOpen className="sidebar-pin-button-icon" aria-hidden="true" />
            )}
            <span className="sidebar-pin-button-label">{pinLabel}</span>
          </button>
        </div>
      </div>
    </aside>
  );
};
