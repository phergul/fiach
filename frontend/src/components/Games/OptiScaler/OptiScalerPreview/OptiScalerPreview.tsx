import type { OptiScalerPreview as OptiScalerPreviewModel } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './OptiScalerPreview.scss';

interface OptiScalerPreviewProps {
  preview: OptiScalerPreviewModel;
}

export const OptiScalerPreview = ({ preview }: OptiScalerPreviewProps) => {
  return (
    <div className="optiscaler-preview">
      {preview.conflicts.length > 0 && (
        <section className="optiscaler-preview-group optiscaler-preview-group-blocked">
          <h3>Blocking conflicts</h3>
          <ul>{preview.conflicts.map((conflict) => <li key={conflict}>{conflict}</li>)}</ul>
        </section>
      )}
      {preview.drift.length > 0 && (
        <section className="optiscaler-preview-group optiscaler-preview-group-warning">
          <h3>Drifted files</h3>
          <ul>
            {preview.drift.map((drift) => (
              <li key={drift.relativePath}>
                {drift.relativePath}{drift.missing ? ' is missing' : ' has changed'}
              </li>
            ))}
          </ul>
        </section>
      )}
      {preview.warnings.length > 0 && (
        <section className="optiscaler-preview-group optiscaler-preview-group-warning">
          <h3>Warnings</h3>
          <ul>{preview.warnings.map((warning) => <li key={warning}>{warning}</li>)}</ul>
        </section>
      )}
      <section className="optiscaler-preview-group">
        <h3>File operations</h3>
        {preview.operations.length === 0 ? (
          <p>No file operations are planned.</p>
        ) : (
          <ul>
            {preview.operations.map((operation, index) => (
              <li key={`${operation.type}-${operation.targetPath}-${index}`}>
                <strong>{operation.type}</strong> {operation.targetPath}
                {operation.backupPath !== undefined && operation.backupPath !== '' && (
                  <span>Backup: {operation.backupPath}</span>
                )}
              </li>
            ))}
          </ul>
        )}
      </section>
      <section className="optiscaler-preview-group">
        <h3>Configuration changes</h3>
        {preview.configurationChanges.length === 0 ? (
          <p>No configuration changes are planned.</p>
        ) : (
          <ul>
            {preview.configurationChanges.map((change) => <li key={change}>{change}</li>)}
          </ul>
        )}
      </section>
      {preview.request.action === 'uninstall' && (
        <section className="optiscaler-preview-group">
          <h3>Retained after uninstall</h3>
          <p>
            Fiach archives the ownership manifest, OptiScaler settings, and available backup snapshot.
            A chained ReShade runtime is restored to its original proxy filename.
          </p>
        </section>
      )}
    </div>
  );
};
