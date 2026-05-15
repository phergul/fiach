import { Modal } from '@components/Common/Modal/Modal';

import './ConfirmDialog.scss';

interface ConfirmDialogProps {
  cancelLabel?: string;
  confirmLabel?: string;
  confirmTone?: 'danger' | 'default';
  isBusy?: boolean;
  isOpen: boolean;
  message: string;
  onCancel: () => void;
  onConfirm: () => void;
  title: string;
}

export const ConfirmDialog = ({
  cancelLabel = 'Cancel',
  confirmLabel = 'Delete',
  confirmTone = 'danger',
  isBusy = false,
  isOpen,
  message,
  onCancel,
  onConfirm,
  title,
}: ConfirmDialogProps) => {
  return (
    <Modal
      background="surface"
      bodyClassName="confirm-dialog-body"
      describedByID="confirm-dialog-message"
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="confirm-dialog-title"
      onClose={onCancel}
      size="sm"
      title={title}
      footer={(
        <>
          <button className="confirm-dialog-button" disabled={isBusy} onClick={onCancel} type="button">
            {cancelLabel}
          </button>
          <button
            className={
              confirmTone === 'danger'
                ? 'confirm-dialog-button confirm-dialog-button-danger'
                : 'confirm-dialog-button'
            }
            disabled={isBusy}
            onClick={onConfirm}
            type="button"
          >
            {confirmLabel}
          </button>
        </>
      )}
    >
      <p className="confirm-dialog-message" id="confirm-dialog-message">
        {message}
      </p>
    </Modal>
  );
};
