import type { DiagnosticLogEntry } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { LogEntryRow } from '@components/Logs/LogEntryRow/LogEntryRow';

import './LogsTable.scss';

interface LogsTableProps {
  entries: DiagnosticLogEntry[];
  errorMessage: string | null;
  isLoading: boolean;
}

export const LogsTable = ({ entries, errorMessage, isLoading }: LogsTableProps) => {
  if (isLoading) {
    return (
      <div className="logs-table-state">
        <StateBlock title="Loading logs" message="Reading recent diagnostic entries." />
      </div>
    );
  }

  if (errorMessage !== null) {
    return (
      <div className="logs-table-state">
        <StateBlock title="Unable to load logs" message={errorMessage} />
      </div>
    );
  }

  if (entries.length === 0) {
    return (
      <div className="logs-table-state">
        <StateBlock title="No logs to show" message="Change filters or refresh to reload persisted entries." />
      </div>
    );
  }

  return (
    <section className="logs-table" aria-label="Diagnostic logs">
      <div className="logs-table-header" role="row">
        <span>Time</span>
        <span>Level</span>
        <span>Operation</span>
        <span>Message</span>
      </div>
      <div className="logs-table-body">
        {entries.map((entry, index) => (
          <LogEntryRow entry={entry} key={`${entry.Timestamp}-${entry.Level}-${entry.Operation}-${index}`} />
        ))}
      </div>
    </section>
  );
};
