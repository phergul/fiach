import { useState } from 'react';
import { PanelLeftClose, PanelLeftOpen } from 'lucide-react';

import { Navigation } from '@components/Navigation/Navigation';

import './Sidebar.scss';

export const Sidebar = () => {
  const [isPinned, setIsPinned] = useState(false);
  const pinLabel = isPinned ? 'Unpin sidebar' : 'Pin sidebar';

  return (
    <aside className={isPinned ? 'sidebar sidebar-pinned' : 'sidebar'} aria-label="Primary navigation">
      <Navigation />
      <button
        aria-label={pinLabel}
        className="sidebar-pin-button"
        onClick={() => setIsPinned((currentIsPinned) => !currentIsPinned)}
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
    </aside>
  );
};
