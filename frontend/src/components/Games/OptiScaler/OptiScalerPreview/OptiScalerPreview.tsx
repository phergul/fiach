import type { Operation as OptiScalerOperation } from '@bindings/github.com/phergul/fiach/internal/optiscaler/models';
import type { OptiScalerPreview as OptiScalerPreviewModel } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './OptiScalerPreview.scss';

interface OptiScalerPreviewProps {
  preview: OptiScalerPreviewModel;
}

interface PreviewGroupProps {
  items: React.ReactNode[];
  title: string;
  tone?: 'danger' | 'success' | 'warning';
}

const filename = (path: string) => path.split(/[\\/]/).pop() ?? path;

const operationDescription = (operation: OptiScalerOperation) => {
  if (operation.type === 'move') {
    return `${filename(operation.sourcePath ?? '')} → ${filename(operation.targetPath)}`;
  }
  if (operation.type === 'restore') {
    return `${filename(operation.sourcePath ?? '')} → ${filename(operation.targetPath)}`;
  }
  return filename(operation.targetPath);
};

const PreviewGroup = ({ items, title, tone }: PreviewGroupProps) => {
  if (items.length === 0) {
    return null;
  }
  return (
    <section
      className={
        tone === undefined
          ? 'optiscaler-preview-group'
          : `optiscaler-preview-group optiscaler-preview-group-${tone}`
      }
    >
      <h3>{title}</h3>
      <ul>
        {items.map((item, index) => (
          <li key={index}>{item}</li>
        ))}
      </ul>
    </section>
  );
};

export const OptiScalerPreview = ({ preview }: OptiScalerPreviewProps) => {
  const filesToAdd = preview.operations.filter(
    (operation) => operation.type === 'copy' || operation.type === 'adopt',
  );
  const filesToBackup = preview.operations.filter((operation) => operation.backupPath);
  const filesToRemove = preview.operations.filter(
    (operation) =>
      operation.type === 'delete' || operation.type === 'move' || operation.type === 'restore',
  );

  return (
    <div className="optiscaler-preview">
      <PreviewGroup items={preview.conflicts} title="Blocking conflicts" tone="danger" />
      <PreviewGroup
        items={preview.drift.map((drift) => (
          <>
            {drift.relativePath}
            {drift.missing ? ' is missing' : ' has changed'}
          </>
        ))}
        title="Drifted files"
        tone="warning"
      />
      <PreviewGroup items={preview.warnings} title="Warnings" tone="warning" />
      <PreviewGroup
        items={filesToAdd.map((operation) => (
          <>
            <span className="optiscaler-preview-symbol">+</span>
            {operationDescription(operation)}
          </>
        ))}
        title="Files to add"
        tone="success"
      />
      <PreviewGroup
        items={filesToBackup.map((operation) => (
          <>
            <span className="optiscaler-preview-symbol">~</span>
            {filename(operation.targetPath)} → {filename(operation.backupPath ?? '')}
          </>
        ))}
        title="Files to backup"
        tone="warning"
      />
      <PreviewGroup
        items={filesToRemove.map((operation) => (
          <>
            <span className="optiscaler-preview-symbol">−</span>
            {operationDescription(operation)}
          </>
        ))}
        title="Files to remove"
        tone="danger"
      />
      <PreviewGroup items={preview.configurationChanges} title="Configuration changes" />
      {preview.operations.length === 0 && preview.configurationChanges.length === 0 && (
        <section className="optiscaler-preview-group">
          <h3>File operations</h3>
          <p>No file operations are planned.</p>
        </section>
      )}
      {preview.request.action === 'uninstall' && (
        <section className="optiscaler-preview-group">
          <h3>Retained after uninstall</h3>
          <p>
            Fiach archives the ownership manifest, OptiScaler settings, and available backup
            snapshot. A chained ReShade runtime is restored to its original proxy filename.
          </p>
        </section>
      )}
    </div>
  );
};
