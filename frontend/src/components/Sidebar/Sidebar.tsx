import { Brand } from '../Brand/Brand';
import { Navigation } from '../Navigation/Navigation';

import './Sidebar.scss';

export const Sidebar = () => {
  return (
    <aside className="sidebar" aria-label="Primary navigation">
      <Brand />
      <Navigation />
    </aside>
  );
};
