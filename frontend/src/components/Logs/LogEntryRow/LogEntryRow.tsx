import { useState } from 'react';

import { ChevronDown, ChevronRight } from 'lucide-react';

import type { DiagnosticLogEntry } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './LogEntryRow.scss';

interface LogEntryRowProps {
  entry: DiagnosticLogEntry;
}

const levelClassNames: Record<string, string> = {
  debug: 'log-entry-row-level-debug',
  error: 'log-entry-row-level-error',
  info: 'log-entry-row-level-info',
  warn: 'log-entry-row-level-warn',
};

const formatTimestamp = (timestamp: string) => {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) {
    return timestamp;
  }

  return date.toLocaleString(undefined, {
    day: '2-digit',
    hour: '2-digit',
    hour12: false,
    minute: '2-digit',
    month: 'short',
    second: '2-digit',
  });
};

const formatOperation = (operation: string) => {
  return operation.trim() === '' ? 'application' : operation.replace(/_/g, ' ');
};

export const LogEntryRow = ({ entry }: LogEntryRowProps) => {
  const [isExpanded, setIsExpanded] = useState(false);
  const detailEntries = Object.entries(entry.Details as Record<string, string | undefined>)
    .filter(([, value]) => value !== undefined && value.trim() !== '')
    .map(([key, value]) => [key, value ?? ''] as const)
    .sort(([left], [right]) => left.localeCompare(right));
  const hasDetails = detailEntries.length > 0;
  const level = entry.Level.toLowerCase();
  const levelClassName = levelClassNames[level] ?? 'log-entry-row-level-info';

  return (
    <article className="log-entry-row">
      <button
        className="log-entry-row-summary"
        disabled={!hasDetails}
        onClick={() => setIsExpanded((current) => !current)}
        type="button"
      >
        <span className="log-entry-row-time">{formatTimestamp(entry.Timestamp)}</span>
        <span className={`log-entry-row-level ${levelClassName}`}>{entry.Level}</span>
        <span className="log-entry-row-operation">{formatOperation(entry.Operation)}</span>
        <span className="log-entry-row-message">
          {hasDetails && (
            <span className="log-entry-row-toggle" aria-hidden="true">
              {isExpanded ? <ChevronDown /> : <ChevronRight />}
            </span>
          )}
          {entry.Message}
        </span>
      </button>

      {hasDetails && isExpanded && (
        <dl className="log-entry-row-details">
          {detailEntries.map(([key, value]) => (
            <div className="log-entry-row-detail" key={key}>
              <dt>{key}</dt>
              <dd>{value}</dd>
            </div>
          ))}
        </dl>
      )}
    </article>
  );
};
