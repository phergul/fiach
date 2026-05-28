import { Braces, ClipboardCopy, Download, RotateCw, Trash2 } from 'lucide-react';

import './LogsToolbar.scss';

export type LogLevelFilter = 'all' | 'debug' | 'info' | 'warn' | 'error';
export type LogOperationFilter = 'all' | 'scan_games' | 'import_mod' | 'apply_profile' | 'restore_vanilla';

interface LogsToolbarProps {
  isExporting: boolean;
  isLoading: boolean;
  isRawJsonVisible: boolean;
  level: LogLevelFilter;
  operation: LogOperationFilter;
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

const operationOptions: Array<{ label: string; value: LogOperationFilter }> = [
  { label: 'All operations', value: 'all' },
  { label: 'Scan games', value: 'scan_games' },
  { label: 'Import mod', value: 'import_mod' },
  { label: 'Apply profile', value: 'apply_profile' },
  { label: 'Restore vanilla', value: 'restore_vanilla' },
];

export const LogsToolbar = ({
  isExporting,
  isLoading,
  isRawJsonVisible,
  level,
  operation,
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
          <select value={level} onChange={(event) => onLevelChange(event.target.value as LogLevelFilter)}>
            {levelOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        <label className="logs-toolbar-field">
          <select
            value={operation}
            onChange={(event) => onOperationChange(event.target.value as LogOperationFilter)}
          >
            {operationOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
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
          <button disabled={isExporting} onClick={onExport} title="Export visible logs" type="button">
            <Download aria-hidden="true" />
          </button>
        </div>
      </div>
    </header>
  );
};
