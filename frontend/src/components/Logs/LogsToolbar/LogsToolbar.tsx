import { Braces, ClipboardCopy, Download, RotateCw, Trash2 } from 'lucide-react';

import './LogsToolbar.scss';

export type LogLevelFilter = 'all' | 'debug' | 'info' | 'warn' | 'error';
export type LogOperationFilter = string;

export interface LogOperationOption {
  label: string;
  value: LogOperationFilter;
}

export interface LogOperationOptionGroup {
  area: string;
  options: LogOperationOption[];
}

interface LogsToolbarProps {
  isExporting: boolean;
  isLoading: boolean;
  isRawJsonVisible: boolean;
  level: LogLevelFilter;
  operation: LogOperationFilter;
  operationOptionGroups: LogOperationOptionGroup[];
  visibleCount: number;
  onClear: () => void;
  onCopy: () => void;
  onExport: () => void;
  onLevelChange: (level: LogLevelFilter) => void;
  onOperationChange: (operation: LogOperationFilter) => void;
  onRawJsonVisibleChange: (isRawJsonVisible: boolean) => void;
  onRefresh: () => void;
}

const levelOptions: Array<{ label: string; value: LogLevelFilter }> = [
  { label: 'All levels', value: 'all' },
  { label: 'Debug', value: 'debug' },
  { label: 'Info', value: 'info' },
  { label: 'Warn', value: 'warn' },
  { label: 'Error', value: 'error' },
];

export const LogsToolbar = ({
  isExporting,
  isLoading,
  isRawJsonVisible,
  level,
  operation,
  operationOptionGroups,
  visibleCount,
  onClear,
  onCopy,
  onExport,
  onLevelChange,
  onOperationChange,
  onRawJsonVisibleChange,
  onRefresh,
}: LogsToolbarProps) => {
  return (
    <header className="logs-toolbar">
      <div className="logs-toolbar-title">
        <h1>Logs</h1>
        <span>{visibleCount} visible</span>
      </div>

      <div className="logs-toolbar-controls">
        <label className="logs-toolbar-field">
          <select
            value={level}
            onChange={(event) => onLevelChange(event.target.value as LogLevelFilter)}
          >
            {levelOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        <label className="logs-toolbar-field logs-toolbar-field-operations">
          <select
            value={operation}
            onChange={(event) => onOperationChange(event.target.value as LogOperationFilter)}
          >
            <option value="all">All operations</option>
            {operationOptionGroups.map((group) => (
              <optgroup key={group.area} label={group.area.toUpperCase()}>
                {group.options.map((option) => (
                  <option key={option.value} value={option.value}>
                    {option.label}
                  </option>
                ))}
              </optgroup>
            ))}
          </select>
        </label>

        <div className="logs-toolbar-actions">
          <button
            aria-pressed={isRawJsonVisible}
            className={isRawJsonVisible ? 'logs-toolbar-action-active' : undefined}
            onClick={() => onRawJsonVisibleChange(!isRawJsonVisible)}
            title={isRawJsonVisible ? 'Show formatted logs' : 'Show raw JSON'}
            type="button"
          >
            <Braces aria-hidden="true" />
          </button>
          <button disabled={isLoading} onClick={onRefresh} title="Refresh logs" type="button">
            <RotateCw aria-hidden="true" />
          </button>
          <button onClick={onCopy} title="Copy visible logs" type="button">
            <ClipboardCopy aria-hidden="true" />
          </button>
          <button onClick={onClear} title="Clear visible logs" type="button">
            <Trash2 aria-hidden="true" />
          </button>
          <button
            disabled={isExporting}
            onClick={onExport}
            title="Export visible logs"
            type="button"
          >
            <Download aria-hidden="true" />
          </button>
        </div>
      </div>
    </header>
  );
};
