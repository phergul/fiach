import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import { Clipboard, Events } from '@wailsio/runtime';
import { ClipboardCopy, RotateCw, Trash2 } from 'lucide-react';

import {
  ClearDevLogs,
  ListDevLogs,
} from '@bindings/github.com/phergul/fiach/internal/services/devservice';
import { DevLogEntry } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { StateBlock } from '@components/Common/StateBlock/StateBlock';
import { TitleBar } from '@components/Common/TitleBar/TitleBar';
import { useToast } from '@components/Common/Toast/Toast';

import './DevLogsWindow.scss';

const maxDevLogEntries = 500;
const logEntryEventName = 'dev:log-entry';

const normalizeLogEntry = (entry: unknown): DevLogEntry => {
  return DevLogEntry.createFrom(entry);
};

const formatLogLine = (entry: DevLogEntry): string => {
  const timestamp = entry.Timestamp.trim();
  const message = entry.Message.trim();

  if (timestamp === '') {
    return message;
  }

  return `${timestamp}  ${message}`;
};

const formatRawLogs = (entries: DevLogEntry[]): string => {
  return entries.map(formatLogLine).join('\n');
};

export const DevLogsWindow = () => {
  const { addErrorToast, addToast } = useToast();
  const [entries, setEntries] = useState<DevLogEntry[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const logContainerRef = useRef<HTMLPreElement>(null);

  const rawLogs = useMemo(() => formatRawLogs(entries), [entries]);

  const loadLogs = useCallback(async () => {
    setIsLoading(true);

    try {
      const logs = await ListDevLogs(maxDevLogEntries);
      setEntries(logs);
    } catch (error) {
      addErrorToast(error);
    } finally {
      setIsLoading(false);
    }
  }, [addErrorToast]);

  useEffect(() => {
    void loadLogs();
  }, [loadLogs]);

  useEffect(() => {
    return Events.On(logEntryEventName, (event) => {
      const entry = normalizeLogEntry(event.data);
      setEntries((currentEntries) => [...currentEntries, entry].slice(-maxDevLogEntries));
    });
  }, []);

  useEffect(() => {
    if (logContainerRef.current === null) {
      return;
    }

    logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
  }, [rawLogs]);

  const copyLogs = async () => {
    if (entries.length === 0) {
      addToast({ message: 'No logs to copy.', tone: 'info' });
      return;
    }

    try {
      await Clipboard.SetText(rawLogs);
      addToast({ message: 'Dev logs copied.', tone: 'success' });
    } catch (error) {
      addErrorToast(error);
    }
  };

  const clearLogs = async () => {
    try {
      await ClearDevLogs();
      setEntries([]);
    } catch (error) {
      addErrorToast(error);
    }
  };

  return (
    <main className="dev-logs-window">
      <TitleBar title="Dev Logs" />
      <header className="dev-logs-toolbar">
        <div className="dev-logs-toolbar-title">
          <h1>Dev Logs</h1>
          <span>{entries.length} logs</span>
        </div>

        <div className="dev-logs-toolbar-actions">
          <button
            disabled={isLoading}
            onClick={() => void loadLogs()}
            title="Refresh logs"
            type="button"
          >
            <RotateCw aria-hidden="true" />
          </button>
          <button
            disabled={entries.length === 0}
            onClick={() => void copyLogs()}
            title="Copy logs"
            type="button"
          >
            <ClipboardCopy aria-hidden="true" />
          </button>
          <button
            disabled={entries.length === 0}
            onClick={() => void clearLogs()}
            title="Clear logs"
            type="button"
          >
            <Trash2 aria-hidden="true" />
          </button>
        </div>
      </header>

      {isLoading ? (
        <div className="dev-logs-state">
          <StateBlock title="Loading logs" message="Reading recent dev log entries." />
        </div>
      ) : entries.length === 0 ? (
        <div className="dev-logs-state">
          <StateBlock title="No logs to show" />
        </div>
      ) : (
        <section className="dev-logs-raw" aria-label="Dev logs">
          <pre ref={logContainerRef}>{rawLogs}</pre>
        </section>
      )}
    </main>
  );
};
