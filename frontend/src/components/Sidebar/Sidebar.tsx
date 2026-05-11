import { Brand } from '@components/Brand/Brand';
import { Navigation } from '@components/Navigation/Navigation';

import './Sidebar.scss';

export const Sidebar = () => {
  return (
    <aside className="sidebar" aria-label="Primary navigation">
      <Brand />
      <Navigation />
    </aside>
  );
};
