import { FolderOpen, RotateCcw } from 'lucide-react';

import './ModStorageRootPathControl.scss';

interface ModStorageRootPathControlProps {
  isBusy: boolean;
  onChooseFolder: () => void;
  onClear: () => void;
  value: string;
}

export const ModStorageRootPathControl = ({
  isBusy,
  onChooseFolder,
  onClear,
  value,
}: ModStorageRootPathControlProps) => {
  const hasValue = value.trim() !== '';
  const valueClassName = hasValue
    ? 'mod-storage-root-path-control-value'
    : 'mod-storage-root-path-control-value mod-storage-root-path-control-value-empty';

  return (
    <div className="mod-storage-root-path-control">
      <div className={valueClassName}>
        {hasValue ? value : 'Not set'}
      </div>

      <div className="mod-storage-root-path-control-actions">
        <button
          className="mod-storage-root-path-control-button"
          disabled={isBusy}
          onClick={onChooseFolder}
          type="button"
        >
          <FolderOpen className="mod-storage-root-path-control-button-icon" aria-hidden="true" />
          Choose Folder
        </button>
        <button
          className="mod-storage-root-path-control-button"
          disabled={isBusy || !hasValue}
          onClick={onClear}
          type="button"
        >
          <RotateCcw className="mod-storage-root-path-control-button-icon" aria-hidden="true" />
          Clear
        </button>
      </div>
    </div>
  );
};
