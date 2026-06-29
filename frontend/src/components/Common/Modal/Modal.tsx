import type { FormEventHandler, ReactNode } from 'react';

import { X } from 'lucide-react';

import './Modal.scss';

type ModalBackground = 'background' | 'surface';
type ModalSize = 'sm' | 'md' | 'lg';

interface ModalProps {
  abovePanel?: ReactNode;
  background?: ModalBackground;
  bodyClassName?: string;
  children: ReactNode;
  closeTitle?: string;
  description?: ReactNode;
  footer?: ReactNode;
  isBusy?: boolean;
  isOpen: boolean;
  labelledByID: string;
  onClose: () => void;
  onSubmit?: FormEventHandler<HTMLFormElement>;
  panelClassName?: string;
  size?: ModalSize;
  title: ReactNode;
  describedByID?: string;
}

export const Modal = ({
  abovePanel,
  background = 'background',
  bodyClassName,
  children,
  closeTitle = 'Close',
  description,
  footer,
  isBusy = false,
  isOpen,
  labelledByID,
  onClose,
  onSubmit,
  panelClassName,
  size = 'md',
  title,
  describedByID,
}: ModalProps) => {
  if (!isOpen) {
    return null;
  }

  const handleClose = () => {
    if (!isBusy) {
      onClose();
    }
  };
  const PanelElement = onSubmit === undefined ? 'section' : 'form';
  const panelClassNames = [
    'modal-panel',
    `modal-panel-${size}`,
    `modal-panel-${background}`,
    panelClassName,
  ]
    .filter(Boolean)
    .join(' ');
  const bodyClassNames = ['modal-body', bodyClassName].filter(Boolean).join(' ');

  return (
    <div className="modal" role="presentation">
      <div className="modal-backdrop" onClick={handleClose} aria-hidden="true" />

      <div
        className={
          abovePanel === undefined ? 'modal-stack' : 'modal-stack modal-stack-with-accessory'
        }
      >
        {abovePanel}

        <PanelElement
          className={panelClassNames}
          onSubmit={onSubmit}
          role="dialog"
          aria-modal="true"
          aria-labelledby={labelledByID}
          aria-describedby={describedByID}
        >
          <header className="modal-header">
            <div className="modal-heading">
              <h2 className="modal-title" id={labelledByID}>
                {title}
              </h2>
              {description !== undefined && <div className="modal-summary">{description}</div>}
            </div>

            <button
              className="modal-close"
              disabled={isBusy}
              onClick={handleClose}
              title={closeTitle}
              type="button"
            >
              <X className="modal-icon" aria-hidden="true" />
            </button>
          </header>

          <div className={bodyClassNames}>{children}</div>

          {footer !== undefined && <footer className="modal-footer">{footer}</footer>}
        </PanelElement>
      </div>
    </div>
  );
};
