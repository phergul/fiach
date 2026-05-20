import { BookOpen, ScrollText, Settings, Users } from 'lucide-react';
import { Fragment } from 'react';
import { NavLink } from 'react-router-dom';

import './Navigation.scss';

const navigationItems = [
  { Icon: BookOpen, label: 'Library', path: '/library' },
  { Icon: Users, label: 'Profiles', path: '/profiles' },
  { Icon: Settings, label: 'Settings', path: '/settings' },
  { Icon: ScrollText, label: 'Logs', path: '/logs' },
];

export const Navigation = () => {
  return (
    <nav className="navigation" aria-label="Main sections">
      {navigationItems.map((item, index) => (
        <Fragment key={item.path}>
          {index > 0 && <span className="navigation-separator" aria-hidden="true" />}
          <NavLink
            className={({ isActive }) =>
              isActive ? 'navigation-link navigation-link-active' : 'navigation-link'
            }
            onClick={(event) => {
              if (event.detail > 0) {
                event.currentTarget.blur();
              }
            }}
            title={item.label}
            to={item.path}
          >
            <item.Icon className="navigation-link-icon" aria-hidden="true" />
            <span className="navigation-link-label">{item.label}</span>
          </NavLink>
        </Fragment>
      ))}
    </nav>
  );
};
