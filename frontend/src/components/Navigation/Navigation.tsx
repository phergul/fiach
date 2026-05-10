import { NavLink } from 'react-router-dom';

import './Navigation.scss';

const navigationItems = [
  { label: 'Library', path: '/library' },
  { label: 'Profiles', path: '/profiles' },
  { label: 'Settings', path: '/settings' },
  { label: 'Logs', path: '/logs' },
];

export const Navigation = () => {
  return (
    <nav className="navigation" aria-label="Main sections">
      {navigationItems.map((item) => (
        <NavLink
          className={({ isActive }) =>
            isActive ? 'navigation-link navigation-link-active' : 'navigation-link'
          }
          key={item.path}
          to={item.path}
        >
          {item.label}
        </NavLink>
      ))}
    </nav>
  );
};
