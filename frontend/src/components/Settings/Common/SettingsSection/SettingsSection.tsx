import type { ReactNode } from 'react';
import { useId } from 'react';

import './SettingsSection.scss';

interface SettingsSectionProps {
  children: ReactNode;
  title: string;
}

export const SettingsSection = ({ children, title }: SettingsSectionProps) => {
  const titleID = useId();

  return (
    <section className="settings-section" aria-labelledby={titleID}>
      <header className="settings-section-header">
        <h2 className="settings-section-title" id={titleID}>
          {title}
        </h2>
      </header>

      <div className="settings-section-content">{children}</div>
    </section>
  );
};
