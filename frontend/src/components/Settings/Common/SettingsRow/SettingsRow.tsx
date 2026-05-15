import type { ReactNode } from 'react';

import './SettingsRow.scss';

interface SettingsRowProps {
  children: ReactNode;
  description?: string;
  status?: string | null;
  title: string;
}

export const SettingsRow = ({ children, description, status = null, title }: SettingsRowProps) => {
  return (
    <div className="settings-row">
      <div className="settings-row-copy">
        <h3 className="settings-row-title">{title}</h3>
        {description !== undefined && <p className="settings-row-description">{description}</p>}
        {status !== null && <p className="settings-row-status">{status}</p>}
      </div>

      <div className="settings-row-control">{children}</div>
    </div>
  );
};
