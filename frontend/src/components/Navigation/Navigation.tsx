import { Fragment, useEffect, useState } from 'react';

import { NavLink } from 'react-router-dom';
import { BookOpen, Bug, ScrollText, Settings } from 'lucide-react';

import { IsDevMode } from '@bindings/github.com/phergul/fiach/internal/services/devservice';
import {
  OpenDevLogsWindow,
  OpenLogsWindow,
} from '@bindings/github.com/phergul/fiach/internal/services/windowservice';

import './Navigation.scss';

const navigationItems = [
  { Icon: BookOpen, label: 'Library', path: '/library' },
  { Icon: Settings, label: 'Settings', path: '/settings' },
];

export const Navigation = () => {
  const [isDevMode, setIsDevMode] = useState(false);

  useEffect(() => {
    IsDevMode().then(setIsDevMode);
  }, []);

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
      <span className="navigation-separator" aria-hidden="true" />
      <button
        className="navigation-link navigation-button"
        onClick={(event) => {
          void OpenLogsWindow();

          if (event.detail > 0) {
            event.currentTarget.blur();
          }
        }}
        title="Logs"
        type="button"
      >
        <ScrollText className="navigation-link-icon" aria-hidden="true" />
        <span className="navigation-link-label">Logs</span>
      </button>
      {isDevMode && (
        <>
          <span className="navigation-separator" aria-hidden="true" />
          <button
            className="navigation-link navigation-button"
            onClick={(event) => {
              void OpenDevLogsWindow();

              if (event.detail > 0) {
                event.currentTarget.blur();
              }
            }}
            title="Dev Logs"
            type="button"
          >
            <Bug className="navigation-link-icon" aria-hidden="true" />
            <span className="navigation-link-label">Dev Logs</span>
          </button>
        </>
      )}
    </nav>
  );
};
