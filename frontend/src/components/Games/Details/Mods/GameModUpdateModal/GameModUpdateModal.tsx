import { FormEvent } from 'react';

import {
  ModSourceType,
  type UpdateModResult,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Modal } from '@components/Common/Modal/Modal';
import { formatModMetadataBytes } from '@components/Games/Details/Mods/ModMetadataSummary/ModMetadataSummary';

import './GameModUpdateModal.scss';

interface GameModUpdateModalProps {
  error: string | null;
  isBusy: boolean;
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => Promise<void> | void;
  result: UpdateModResult | null;
}

type SnapshotValue = string | null;

const sourceTypeLabel = (sourceType: ModSourceType) => {
  return sourceType === ModSourceType.ModSourceTypeArchive ? 'Archive' : 'Folder';
};

const sourceName = (result: UpdateModResult) => {
  return result.After.OriginalSourceName ?? result.After.OriginalSourcePath;
};

const formatCount = (count: number | null) => (count === null ? null : count.toLocaleString());

const formatSize = (bytes: number | null) =>
  bytes === null ? null : formatModMetadataBytes(bytes);

const formatValue = (value: SnapshotValue) => {
  if (value === null || value === '') {
    return 'Not available';
  }

  return String(value);
};

const changedClassName = (before: SnapshotValue, after: SnapshotValue) => {
  return before !== after
    ? 'game-mod-update-modal-comparison-value game-mod-update-modal-comparison-value-changed'
    : 'game-mod-update-modal-comparison-value';
};

const ComparisonRow = ({
  after,
  before,
  label,
}: {
  after: SnapshotValue;
  before: SnapshotValue;
  label: string;
}) => (
  <div className="game-mod-update-modal-comparison-row">
    <span className="game-mod-update-modal-comparison-label">{label}</span>
    <span className="game-mod-update-modal-comparison-current">{formatValue(before)}</span>
    <span className={changedClassName(before, after)}>{formatValue(after)}</span>
  </div>
);

export const GameModUpdateModal = ({
  error,
  isBusy,
  isOpen,
  onClose,
  onConfirm,
  result,
}: GameModUpdateModalProps) => {
  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    await onConfirm();
  };

  const footer = (
    <>
      <button
        className="game-mod-update-modal-primary-button"
        disabled={isBusy || result === null}
        type="submit"
      >
        {isBusy ? 'Updating...' : 'Update Mod'}
      </button>
      <button
        className="game-mod-update-modal-secondary-button"
        disabled={isBusy}
        onClick={onClose}
        type="button"
      >
        Cancel
      </button>
    </>
  );

  return (
    <Modal
      bodyClassName="game-mod-update-modal-body"
      closeTitle="Close update review"
      description="Profile membership, load order, and install configuration will be preserved."
      footer={footer}
      isBusy={isBusy}
      isOpen={isOpen}
      labelledByID="game-mod-update-modal-title"
      onClose={onClose}
      onSubmit={handleSubmit}
      size="lg"
      title="Update Mod"
    >
      {result !== null && (
        <div className="game-mod-update-modal">
          <div className="game-mod-update-modal-summary">
            <div className="game-mod-update-modal-summary-item">
              <span className="game-mod-update-modal-label">Mod</span>
              <span className="game-mod-update-modal-value">{result.Mod.Name}</span>
            </div>
            <div className="game-mod-update-modal-summary-item">
              <span className="game-mod-update-modal-label">Replacement source</span>
              <span className="game-mod-update-modal-path">{sourceName(result)}</span>
            </div>
            <div className="game-mod-update-modal-summary-item game-mod-update-modal-summary-item-wide">
              <span className="game-mod-update-modal-label">Source path</span>
              <span className="game-mod-update-modal-path">{result.After.OriginalSourcePath}</span>
            </div>
          </div>

          <div className="game-mod-update-modal-comparison" aria-label="Package changes">
            <div className="game-mod-update-modal-comparison-header">
              <span>Package</span>
              <span>Current</span>
              <span>Replacement</span>
            </div>
            <ComparisonRow
              after={sourceTypeLabel(result.After.SourceType)}
              before={sourceTypeLabel(result.Before.SourceType)}
              label="Source type"
            />
            <ComparisonRow
              after={formatCount(result.After.FileCount)}
              before={formatCount(result.Before.FileCount)}
              label="Files"
            />
            <ComparisonRow
              after={formatCount(result.After.DirectoryCount)}
              before={formatCount(result.Before.DirectoryCount)}
              label="Folders"
            />
            <ComparisonRow
              after={formatSize(result.After.TotalSizeBytes)}
              before={formatSize(result.Before.TotalSizeBytes)}
              label="Size"
            />
          </div>

          {result.MetadataWarning !== null && (
            <p className="game-mod-update-modal-warning">
              Metadata could not be read from the replacement package. Existing detected metadata
              will be kept.
            </p>
          )}

          {result.Warnings.map((warning) => (
            <p className="game-mod-update-modal-warning" key={warning}>
              {warning}
            </p>
          ))}

          {result.RequiresReapply && (
            <p className="game-mod-update-modal-warning">
              This mod is part of the currently applied profile. Reapply that profile when you want
              the game files to use this update.
            </p>
          )}
        </div>
      )}

      {error !== null && <p className="game-mod-update-modal-error">{error}</p>}
    </Modal>
  );
};
