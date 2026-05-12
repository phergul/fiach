import './ConfirmDialog.scss';

interface ConfirmDialogProps {
  cancelLabel?: string;
  confirmLabel?: string;
  isOpen: boolean;
  message: string;
  onCancel: () => void;
  onConfirm: () => void;
  title: string;
}

export const ConfirmDialog = ({
  cancelLabel = 'Cancel',
  confirmLabel = 'Delete',
  isOpen,
  message,
  onCancel,
  onConfirm,
  title,
}: ConfirmDialogProps) => {
  if (!isOpen) {
    return null;
  }

  return (
    <div className="confirm-dialog" role="presentation">
      <div className="confirm-dialog-backdrop" onClick={onCancel} aria-hidden="true" />
      <section
        className="confirm-dialog-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby="confirm-dialog-title"
        aria-describedby="confirm-dialog-message"
      >
        <h2 className="confirm-dialog-title" id="confirm-dialog-title">
          {title}
        </h2>
        <p className="confirm-dialog-message" id="confirm-dialog-message">
          {message}
        </p>
        <div className="confirm-dialog-actions">
          <button className="confirm-dialog-button" onClick={onCancel} type="button">
            {cancelLabel}
          </button>
          <button
            className="confirm-dialog-button confirm-dialog-button-danger"
            onClick={onConfirm}
            type="button"
          >
            {confirmLabel}
          </button>
        </div>
      </section>
    </div>
  );
};
