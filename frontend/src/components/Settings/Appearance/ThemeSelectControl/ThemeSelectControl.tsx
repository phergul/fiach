import type { ChangeEvent } from 'react';

import type { ThemeDefinition } from '@theme/themeTypes';

import './ThemeSelectControl.scss';

interface ThemeSelectControlProps {
  isBusy: boolean;
  onChange: (themeID: string) => void;
  themes: ThemeDefinition[];
  value: string;
}

export const ThemeSelectControl = ({
  isBusy,
  onChange,
  themes,
  value,
}: ThemeSelectControlProps) => {
  const handleChange = (event: ChangeEvent<HTMLSelectElement>) => {
    onChange(event.target.value);
  };

  return (
    <select
      aria-label="Theme"
      className="theme-select-control"
      disabled={isBusy}
      onChange={handleChange}
      value={value}
    >
      {themes.map((theme) => (
        <option key={theme.id} value={theme.id}>
          {theme.label}
        </option>
      ))}
    </select>
  );
};
