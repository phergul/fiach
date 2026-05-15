import type { ReactNode } from 'react';

import './SettingsPage.scss';

interface SettingsPageProps {
  children: ReactNode;
  title: string;
}

export const SettingsPage = ({ children, title }: SettingsPageProps) => {
  return (
    <section className="settings-page" aria-labelledby="settings-page-title">
      <header className="settings-page-header">
        <h1 className="settings-page-title" id="settings-page-title">
          {title}
        </h1>
      </header>

      <div className="settings-page-content">{children}</div>
    </section>
  );
};
