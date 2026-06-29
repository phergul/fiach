import { Modal } from '@components/Common/Modal/Modal';
import type { ImportQueueItem } from '@hooks';

import './GameModImportQueueSummary.scss';

interface GameModImportQueueSummaryProps {
  counts: {
    failed: number;
    imported: number;
    skipped: number;
  };
  isBusy: boolean;
  isOpen: boolean;
  items: ImportQueueItem[];
  onClose: () => void;
}

const summaryItems = (items: ImportQueueItem[]) =>
  items.filter((item) => item.status === 'failed' || item.status === 'skipped');

export const GameModImportQueueSummary = ({
  counts,
  isBusy,
  isOpen,
  items,
  onClose,
}: GameModImportQueueSummaryProps) => {
  const issues = summaryItems(items);
  const footer = (
    <button className="button-main" disabled={isBusy} onClick={onClose} type="button">
      Done
    </button>
  );

  return (
    <Modal
      bodyClassName="game-mod-import-queue-summary-body"
      closeTitle="Close import summary"
      description="Import queue finished. Review the results below."
      footer={footer}
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-mod-import-queue-summary-title"
      onClose={onClose}
      panelClassName="game-mod-import-queue-summary-panel"
      size="md"
      title="Import Complete"
    >
      <div className="game-mod-import-queue-summary-counts">
        <div className="game-mod-import-queue-summary-count">
          <span>Imported</span>
          <strong>{counts.imported}</strong>
        </div>
        <div className="game-mod-import-queue-summary-count">
          <span>Failed</span>
          <strong>{counts.failed}</strong>
        </div>
        <div className="game-mod-import-queue-summary-count">
          <span>Skipped</span>
          <strong>{counts.skipped}</strong>
        </div>
      </div>

      {issues.length > 0 && (
        <ul className="game-mod-import-queue-summary-list">
          {issues.map((item) => (
            <li className="game-mod-import-queue-summary-item" key={item.id}>
              <strong>{item.initialName}</strong>
              <span>{item.statusMessage ?? item.error ?? item.sourcePath}</span>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  );
};
