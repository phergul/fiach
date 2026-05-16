import { FormEvent, useEffect, useState } from 'react';

import { Modal } from '@components/Common/Modal/Modal';

import './GameModImportReviewDialog.scss';

interface GameModImportReviewDialogProps {
  error: string | null;
  initialName: string;
  isBusy: boolean;
  isOpen: boolean;
  onClose: () => void;
  onImport: (name: string) => Promise<void> | void;
  sourceLabel: string;
  sourcePath: string;
  targetPath: string;
}

export const GameModImportReviewDialog = ({
  error,
  initialName,
  isBusy,
  isOpen,
  onClose,
  onImport,
  sourceLabel,
  sourcePath,
  targetPath,
}: GameModImportReviewDialogProps) => {
  const [name, setName] = useState(initialName);
  const trimmedName = name.trim();

  useEffect(() => {
    if (isOpen) {
      setName(initialName);
    }
  }, [initialName, isOpen]);

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    if (trimmedName === '' || isBusy) {
      return;
    }

    await onImport(trimmedName);
  };

  return (
    <Modal
      bodyClassName="game-mod-import-review-dialog-body"
      closeTitle="Close import review"
      description="Check the name and destination before copying."
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-mod-import-review-dialog-title"
      onClose={onClose}
      onSubmit={handleSubmit}
      title="Review Import"
      footer={(
        <>
          <button
            className="game-mod-import-review-dialog-import-button"
            disabled={isBusy || trimmedName === ''}
            type="submit"
          >
            {isBusy ? 'Importing...' : 'Import Mod'}
          </button>
          <button
            className="game-mod-import-review-dialog-cancel-button"
            disabled={isBusy}
            onClick={onClose}
            type="button"
          >
            Cancel
          </button>
        </>
      )}
    >
      <label className="game-mod-import-review-dialog-field">
        <span className="game-mod-import-review-dialog-label">Mod name</span>
        <input
          className="game-mod-import-review-dialog-input"
          disabled={isBusy}
          onChange={(event) => setName(event.target.value)}
          type="text"
          value={name}
        />
      </label>

      <div className="game-mod-import-review-dialog-paths">
        <div className="game-mod-import-review-dialog-path-row">
          <span className="game-mod-import-review-dialog-label">{sourceLabel}</span>
          <span className="game-mod-import-review-dialog-path">{sourcePath}</span>
        </div>

        <div className="game-mod-import-review-dialog-path-row">
          <span className="game-mod-import-review-dialog-label">Target location</span>
          <span className="game-mod-import-review-dialog-path">{targetPath}</span>
        </div>
      </div>

      {error !== null && <p className="game-mod-import-review-dialog-error">{error}</p>}
    </Modal>
  );
};
