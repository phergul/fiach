import { useState } from 'react';
import { Outlet } from 'react-router-dom';

import { Sidebar } from '@components/Sidebar/Sidebar';

import './Layout.scss';

export const Layout = () => {
  const [isSidebarPinned, setIsSidebarPinned] = useState(false);

  return (
    <div className={isSidebarPinned ? 'layout layout-sidebar-pinned' : 'layout'}>
      <Sidebar isPinned={isSidebarPinned} onPinnedChange={setIsSidebarPinned} />

      <main className="layout-main">
        <div className="layout-main-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
};
