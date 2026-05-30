import { useCallback, useEffect, useMemo, useState } from 'react';

import { Clipboard, Dialogs, Events } from '@wailsio/runtime';

import {
  ExportLogs,
  ListRecentLogs,
  ListRecentRawLogs,
} from '@bindings/github.com/phergul/fiach/internal/services/diagnosticsservice';
import {
  DiagnosticLogEntry,
  ExportDiagnosticLogsInput,
  ListDiagnosticLogsInput,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { useToast } from '@components/Common/Toast/Toast';
import { LogsTable } from '@components/Logs/LogsTable/LogsTable';
import { LogsToolbar, LogLevelFilter, LogOperationFilter } from '@components/Logs/LogsToolbar/LogsToolbar';
import { getErrorMessage } from '@utils';

import './LogsWindow.scss';

const maxRecentLogEntries = 500;
const logEntryEventName = 'diagnostics:log-entry';

const normalizeLogEntry = (entry: unknown): DiagnosticLogEntry => {
  return DiagnosticLogEntry.createFrom(entry);
};

const matchesLevel = (entry: DiagnosticLogEntry, level: LogLevelFilter) => {
  return level === 'all' || entry.Level.toLowerCase() === level;
};

const matchesOperation = (entry: DiagnosticLogEntry, operation: LogOperationFilter) => {
  return operation === 'all' || entry.Operation === operation;
};

const formatLogEntries = (entries: DiagnosticLogEntry[]) => {
  return entries
    .map((entry) => {
      const header = [
        entry.Timestamp.trim(),
        entry.Level.trim().toUpperCase(),
        entry.Operation.trim() === '' ? '' : `[${entry.Operation.trim()}]`,
        entry.Message.trim(),
      ]
        .filter(Boolean)
        .join(' ');

      const details = Object.entries(entry.Details as Record<string, string | undefined>)
        .filter(([, value]) => value !== undefined && value.trim() !== '')
        .sort(([left], [right]) => left.localeCompare(right))
        .map(([key, value]) => `  ${key}: ${value}`)
        .join('\n');

      return details === '' ? header : `${header}\n${details}`;
    })
    .join('\n');
};

export const LogsWindow = () => {
  const { addToast } = useToast();
  const [entries, setEntries] = useState<DiagnosticLogEntry[]>([]);
  const [level, setLevel] = useState<LogLevelFilter>('all');
  const [operation, setOperation] = useState<LogOperationFilter>('all');
  const [isLoading, setIsLoading] = useState(true);
  const [isRawJsonVisible, setIsRawJsonVisible] = useState(false);
  const [rawJson, setRawJson] = useState('[]');
  const [rawJsonIsLoading, setRawJsonIsLoading] = useState(false);
  const [rawJsonErrorMessage, setRawJsonErrorMessage] = useState<string | null>(null);
  const [isExporting, setIsExporting] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const visibleEntries = useMemo(
    () => entries.filter((entry) => matchesLevel(entry, level) && matchesOperation(entry, operation)),
    [entries, level, operation],
  );

  const loadLogs = useCallback(async () => {
    setIsLoading(true);
    setErrorMessage(null);

    try {
      const recentEntries = await ListRecentLogs(
        new ListDiagnosticLogsInput({
          Limit: maxRecentLogEntries,
        }),
      );
      setEntries(recentEntries ?? []);
    } catch (error) {
      const message = getErrorMessage(error);
      setErrorMessage(message);
      addToast({ message, tone: 'error' });
    } finally {
      setIsLoading(false);
    }
  }, [addToast]);

  const loadRawLogs = useCallback(async () => {
    setRawJsonIsLoading(true);
    setRawJsonErrorMessage(null);

    try {
      const operationFilter = operation === 'all' ? '' : operation;
      const levelFilter = level === 'all' ? '' : level;
      const content = await ListRecentRawLogs(
        new ListDiagnosticLogsInput({
          Limit: maxRecentLogEntries,
          Operation: operationFilter,
          Level: levelFilter,
        }),
      );
      setRawJson(content.trim() === '' ? '[]' : content);
    } catch (error) {
      const message = getErrorMessage(error);
      setRawJsonErrorMessage(message);
      addToast({ message, tone: 'error' });
    } finally {
      setRawJsonIsLoading(false);
    }
  }, [addToast, level, operation]);

  useEffect(() => {
    void loadLogs();
  }, [loadLogs]);

  useEffect(() => {
    if (!isRawJsonVisible) {
      return;
    }

    void loadRawLogs();
  }, [isRawJsonVisible, loadRawLogs]);

  useEffect(() => {
    return Events.On(logEntryEventName, (event) => {
      const entry = normalizeLogEntry(event.data);
      setEntries((currentEntries) => [entry, ...currentEntries].slice(0, maxRecentLogEntries));
    });
  }, []);

  const copyVisibleLogs = async () => {
    if (visibleEntries.length === 0) {
      addToast({ message: 'No visible logs to copy.', tone: 'info' });
      return;
    }

    try {
      await Clipboard.SetText(formatLogEntries(visibleEntries));
      addToast({ message: 'Visible logs copied.', tone: 'success' });
    } catch (error) {
      addToast({ message: getErrorMessage(error), tone: 'error' });
    }
  };

  const clearVisibleLogs = () => {
    setEntries((currentEntries) =>
      currentEntries.filter((entry) => !(matchesLevel(entry, level) && matchesOperation(entry, operation))),
    );
    setErrorMessage(null);
  };

  const exportVisibleLogs = async () => {
    if (visibleEntries.length === 0) {
      addToast({ message: 'No visible logs to export.', tone: 'info' });
      return;
    }

    const path = await Dialogs.SaveFile({
      ButtonText: 'Export',
      CanCreateDirectories: true,
      Filename: 'fiach-logs.txt',
      Filters: [
        {
          DisplayName: 'Text Files',
          Pattern: '*.txt',
        },
      ],
      Title: 'Export Logs',
    });

    if (path.trim() === '') {
      return;
    }

    setIsExporting(true);
    try {
      await ExportLogs(
        new ExportDiagnosticLogsInput({
          Path: path,
          Entries: visibleEntries,
        }),
      );
      addToast({ message: 'Visible logs exported.', tone: 'success' });
    } catch (error) {
      addToast({ message: getErrorMessage(error), tone: 'error' });
    } finally {
      setIsExporting(false);
    }
  };

  return (
    <main className="logs-window">
      <LogsToolbar
        isExporting={isExporting}
        isLoading={isLoading}
        isRawJsonVisible={isRawJsonVisible}
        level={level}
        operation={operation}
        visibleCount={visibleEntries.length}
        onClear={clearVisibleLogs}
        onCopy={() => void copyVisibleLogs()}
        onExport={() => void exportVisibleLogs()}
        onLevelChange={setLevel}
        onOperationChange={setOperation}
        onRawJsonVisibleChange={setIsRawJsonVisible}
        onRefresh={() => {
          void loadLogs();
          if (isRawJsonVisible) {
            void loadRawLogs();
          }
        }}
      />
      <LogsTable
        entries={visibleEntries}
        errorMessage={errorMessage}
        isLoading={isLoading}
        isRawJsonVisible={isRawJsonVisible}
        rawJson={rawJson}
        rawJsonErrorMessage={rawJsonErrorMessage}
        rawJsonIsLoading={rawJsonIsLoading}
      />
    </main>
  );
};
