import type { Operation as ReShadeOperation, PathImpact, Preview as ReShadePreviewModel } from '@bindings/github.com/phergul/fiach/internal/reshade/models';
import type { ReShadeChainTarget } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './ReShadePreview.scss';

interface ReShadePreviewProps {
  chainTarget: ReShadeChainTarget | null;
  preview: ReShadePreviewModel;
}

interface PreviewGroupProps {
  items: React.ReactNode[];
  title: string;
  tone?: 'danger' | 'success' | 'warning';
}

const filename = (path: string) => path.split(/[\\/]/).pop() ?? path;

const operationDescription = (operation: ReShadeOperation) => {
  if (operation.type === 'copy' || operation.type === 'restore' || operation.type === 'move') {
    return `${filename(operation.sourcePath ?? '')} -> ${filename(operation.targetPath)}`;
  }
  return filename(operation.targetPath);
};

const PreviewGroup = ({ items, title, tone }: PreviewGroupProps) => {
  if (items.length === 0) {
    return null;
  }
  return (
    <section className={tone === undefined
      ? 'reshade-preview-group'
      : `reshade-preview-group reshade-preview-group-${tone}`}
    >
      <h3>{title}</h3>
      <ul>{items.map((item, index) => <li key={index}>{item}</li>)}</ul>
    </section>
  );
};

const impactsByRole = (impacts: PathImpact[], roles: string[]) =>
  impacts.filter((impact) => roles.includes(impact.role));

const pathImpactLabel = (impact: PathImpact) =>
  `${impact.action}: ${impact.path}${impact.preservationOnly ? ' (preserve)' : ''}`;

const filesToAdd = (operations: ReShadeOperation[]) =>
  operations.filter((operation) => operation.type === 'copy' || operation.type === 'adopt');

const filesToRemove = (operations: ReShadeOperation[]) =>
  operations.filter((operation) => operation.type === 'delete' || operation.type === 'move' || operation.type === 'restore');

const chainItems = (chainTarget: ReShadeChainTarget | null) => {
  if (chainTarget === null) {
    return [];
  }
  return [
    `${chainTarget.PrimaryOwner} owns ${chainTarget.PrimaryProxyFilename}`,
    ...(chainTarget.OptiScaler !== null ? [`OptiScaler: ${chainTarget.OptiScaler.ProxyFilename}`] : []),
    ...(chainTarget.ReShade !== null ? [`ReShade: ${chainTarget.ReShade.ActiveRuntimeFilename}`] : []),
  ];
};

export const ReShadePreview = ({ chainTarget, preview }: ReShadePreviewProps) => {
  const runtimeImpacts = impactsByRole(preview.pathImpacts, ['runtime']);
  const configurationImpacts = impactsByRole(preview.pathImpacts, ['configuration', 'preset']);
  const contentImpacts = impactsByRole(preview.pathImpacts, ['effects', 'textures', 'addons']);
  const backupImpacts = impactsByRole(preview.pathImpacts, ['backup']);

  return (
    <div className="reshade-preview">
      <PreviewGroup items={preview.conflicts} title="Blocking conflicts" tone="danger" />
      <PreviewGroup
        items={preview.drift.map((drift) => `${drift.relativePath}${drift.missing ? ' is missing' : ' has changed'}`)}
        title="Managed file drift"
        tone="warning"
      />
      <PreviewGroup
        items={preview.userContentDrift.map((drift) => `${drift.path}${drift.missing ? ' is missing' : ' has changed'}`)}
        title="User content drift"
        tone="warning"
      />
      <PreviewGroup items={preview.warnings} title="Warnings" tone="warning" />
      <PreviewGroup
        items={runtimeImpacts.map(pathImpactLabel)}
        title="Runtime files"
      />
      <PreviewGroup
        items={configurationImpacts.map(pathImpactLabel)}
        title="Configuration and presets"
      />
      <PreviewGroup
        items={contentImpacts.map(pathImpactLabel)}
        title="Effects, textures, and add-ons"
      />
      <PreviewGroup
        items={backupImpacts.map(pathImpactLabel)}
        title="Backups"
        tone="warning"
      />
      <PreviewGroup
        items={filesToAdd(preview.operations).map((operation) => (
          <><span className="reshade-preview-symbol">+</span>{operationDescription(operation)}</>
        ))}
        title="Files to add or replace"
        tone="success"
      />
      <PreviewGroup
        items={filesToRemove(preview.operations).map((operation) => (
          <><span className="reshade-preview-symbol">-</span>{operationDescription(operation)}</>
        ))}
        title="Files to remove or restore"
        tone="danger"
      />
      <PreviewGroup items={chainItems(chainTarget)} title="Injection chain" />
      {preview.operations.length === 0 && preview.pathImpacts.length === 0 && (
        <section className="reshade-preview-group">
          <h3>File operations</h3>
          <p>No file operations are planned.</p>
        </section>
      )}
    </div>
  );
};
