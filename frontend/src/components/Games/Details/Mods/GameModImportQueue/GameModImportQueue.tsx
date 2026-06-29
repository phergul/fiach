import { Modal } from '@components/Common/Modal/Modal';
import type { ImportQueueItem } from '@hooks';

import './GameModImportQueue.scss';

interface GameModImportQueueProps {
  isBusy: boolean;
  isOpen: boolean;
  items: ImportQueueItem[];
  onClose: () => void;
  onRemoveItem: (itemID: string) => void;
  onReviewItem: (itemID: string) => void;
  onSkipItem: (itemID: string) => void;
}

const statusLabel = (item: ImportQueueItem) => {
  switch (item.status) {
    case 'pending':
      return 'Pending';
    case 'reviewing':
      return 'Reviewing';
    case 'importing':
      return 'Importing';
    case 'imported':
      return 'Imported';
    case 'failed':
      return 'Failed';
    case 'skipped':
      return 'Skipped';
    default:
      return item.status;
  }
};

const itemDetail = (item: ImportQueueItem) => {
  if (item.status === 'imported' && item.importedModName !== undefined) {
    return item.importedModName;
  }

  if (item.statusMessage !== undefined && item.statusMessage !== '') {
    return item.statusMessage;
  }

  if (item.error !== undefined && item.error !== '') {
    return item.error;
  }

  return item.sourcePath;
};

export const GameModImportQueue = ({
  isBusy,
  isOpen,
  items,
  onClose,
  onRemoveItem,
  onReviewItem,
  onSkipItem,
}: GameModImportQueueProps) => {
  const footer = (
    <div className="game-mod-import-queue-footer">
      <button
        className="game-mod-import-queue-close-button"
        disabled={isBusy}
        onClick={onClose}
        type="button"
      >
        Close
      </button>
    </div>
  );

  return (
    <Modal
      bodyClassName="game-mod-import-queue-body"
      closeTitle="Close import queue"
      description="Review, skip, or remove queued imports before continuing."
      footer={footer}
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-mod-import-queue-title"
      onClose={onClose}
      panelClassName="game-mod-import-queue-panel"
      size="lg"
      title="Import Queue"
    >
      <div className="game-mod-import-queue-list-body">
        <ul className="game-mod-import-queue-list">
          {items.map((item) => (
            <li className="game-mod-import-queue-list-item" key={item.id}>
              <div className="game-mod-import-queue-item">
                <div className="game-mod-import-queue-item-copy">
                  <span
                    className={`game-mod-import-queue-status game-mod-import-queue-status-${item.status}`}
                  >
                    {statusLabel(item)}
                  </span>
                  <span className="game-mod-import-queue-item-name">{item.initialName}</span>
                  <span className="game-mod-import-queue-item-detail">{itemDetail(item)}</span>
                </div>

                <div className="game-mod-import-queue-item-actions">
                  {item.status === 'pending' && (
                    <>
                      <button disabled={isBusy} onClick={() => onReviewItem(item.id)} type="button">
                        Review
                      </button>
                      <button disabled={isBusy} onClick={() => onSkipItem(item.id)} type="button">
                        Skip
                      </button>
                    </>
                  )}
                  {(item.status === 'pending' ||
                    item.status === 'skipped' ||
                    item.status === 'failed') && (
                    <button disabled={isBusy} onClick={() => onRemoveItem(item.id)} type="button">
                      Remove
                    </button>
                  )}
                  {item.status === 'failed' && (
                    <button disabled={isBusy} onClick={() => onReviewItem(item.id)} type="button">
                      Retry
                    </button>
                  )}
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </Modal>
  );
};
