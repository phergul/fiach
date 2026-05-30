import { createContext, ReactNode, useCallback, useContext, useMemo, useState } from 'react';
import { X } from 'lucide-react';

import { OpenLogsWindow } from '@bindings/github.com/phergul/fiach/internal/services/windowservice';
import { getErrorMessage } from '@utils';

import './Toast.scss';

type ToastTone = 'error' | 'info' | 'success';

interface ToastAction {
  label: string;
  onSelect: () => void;
}

interface ToastMessage {
  action?: ToastAction;
  id: string;
  message: string;
  tone: ToastTone;
}

interface AddToastOptions {
  action?: ToastAction;
  duration?: number;
  message: string;
  tone?: ToastTone;
}

interface ToastContextValue {
  addToast: (options: AddToastOptions) => string;
  addErrorToast: (error: unknown, options?: Pick<AddToastOptions, 'duration'>) => string;
  removeToast: (id: string) => void;
}

interface ToastProviderProps {
  children: ReactNode;
}

const toastDefaultDurations: Record<ToastTone, number> = {
  error: 12000,
  info: 6400,
  success: 6400,
};
const ToastContext = createContext<ToastContextValue | null>(null);

export const ToastProvider = ({ children }: ToastProviderProps) => {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((currentToasts) => currentToasts.filter((toast) => toast.id !== id));
  }, []);

  const addToast = useCallback(
    ({ action, duration, message, tone = 'info' }: AddToastOptions) => {
      const id = crypto.randomUUID();
      const toastDuration = duration ?? toastDefaultDurations[tone];

      setToasts((currentToasts) => [...currentToasts, { action, id, message, tone }]);

      if (toastDuration > 0) {
        window.setTimeout(() => removeToast(id), toastDuration);
      }

      return id;
    },
    [removeToast],
  );

  const addErrorToast = useCallback(
    (error: unknown, options: Pick<AddToastOptions, 'duration'> = {}) =>
      addToast({
        action: {
          label: 'Logs',
          onSelect: () => {
            void OpenLogsWindow();
          },
        },
        duration: options.duration,
        message: getErrorMessage(error),
        tone: 'error',
      }),
    [addToast],
  );

  const value = useMemo(
    () => ({
      addToast,
      addErrorToast,
      removeToast,
    }),
    [addErrorToast, addToast, removeToast],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast" aria-live="polite" aria-relevant="additions">
        {toasts.map((toast) => (
          <div className={`toast-message toast-message-${toast.tone}`} key={toast.id}>
            <p className="toast-message-text">{toast.message}</p>
            <div className="toast-message-controls">
              {toast.action !== undefined && (
                <button
                  className="toast-message-action"
                  onClick={toast.action.onSelect}
                  type="button"
                >
                  {toast.action.label}
                </button>
              )}
              <button
                aria-label="Dismiss notification"
                className="toast-message-close"
                onClick={() => removeToast(toast.id)}
                title="Dismiss"
                type="button"
              >
                <X className="toast-message-close-icon" aria-hidden="true" />
              </button>
            </div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
};

export const useToast = () => {
  const context = useContext(ToastContext);
  if (context === null) {
    throw new Error('useToast must be used inside ToastProvider');
  }

  return context;
};
