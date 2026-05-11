import { Outlet } from 'react-router-dom';

import { Sidebar } from '@components/Sidebar/Sidebar';

import './Layout.scss';

export const Layout = () => {
  return (
    <div className="layout">
      <Sidebar />

      <main className="layout-main">
        <div className="layout-main-content">
          <Outlet />
        </div>
      </main>
    </div>
  );
};
