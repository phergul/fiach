import { useState } from 'react';
import { Outlet } from 'react-router-dom';

import { TitleBar } from '@components/Common/TitleBar/TitleBar';
import { Sidebar } from '@components/Sidebar/Sidebar';

import './Layout.scss';

export const Layout = () => {
  const [isSidebarPinned, setIsSidebarPinned] = useState(false);

  return (
    <div className="window-shell">
      <TitleBar title="Fiach" />

      <div className="window-shell-body">
        <div className={isSidebarPinned ? 'layout layout-sidebar-pinned' : 'layout'}>
          <Sidebar isPinned={isSidebarPinned} onPinnedChange={setIsSidebarPinned} />

          <main className="layout-main">
            <div className="layout-route">
              <Outlet />
            </div>
          </main>
        </div>
      </div>
    </div>
  );
};
