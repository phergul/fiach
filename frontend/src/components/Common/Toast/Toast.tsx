import { createContext, ReactNode, useCallback, useContext, useMemo, useState } from 'react';

import './Toast.scss';

type ToastTone = 'error' | 'info' | 'success';

interface ToastMessage {
  id: string;
  message: string;
  tone: ToastTone;
}

interface AddToastOptions {
  duration?: number;
  message: string;
  tone?: ToastTone;
}

interface ToastContextValue {
  addToast: (options: AddToastOptions) => string;
  removeToast: (id: string) => void;
}

interface ToastProviderProps {
  children: ReactNode;
}

const toastDefaultDuration = 6400;
const ToastContext = createContext<ToastContextValue | null>(null);

export const ToastProvider = ({ children }: ToastProviderProps) => {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const removeToast = useCallback((id: string) => {
    setToasts((currentToasts) => currentToasts.filter((toast) => toast.id !== id));
  }, []);

  const addToast = useCallback(
    ({ duration = toastDefaultDuration, message, tone = 'info' }: AddToastOptions) => {
      const id = crypto.randomUUID();

      setToasts((currentToasts) => [...currentToasts, { id, message, tone }]);

      if (duration > 0) {
        window.setTimeout(() => removeToast(id), duration);
      }

      return id;
    },
    [removeToast],
  );

  const value = useMemo(
    () => ({
      addToast,
      removeToast,
    }),
    [addToast, removeToast],
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="toast" aria-live="polite" aria-relevant="additions">
        {toasts.map((toast) => (
          <div className={`toast-message toast-message-${toast.tone}`} key={toast.id}>
            {toast.message}
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
