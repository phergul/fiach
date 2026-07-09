import type {
  Operation as ReShadeOperation,
  PathImpact,
  Preview as ReShadePreviewModel,
} from '@bindings/github.com/phergul/fiach/internal/reshade/models';
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

const normalizeSeparators = (path: string) => path.replace(/\\/g, '/');

const displayPath = (path: string, targetRelativePath: string) => {
  const normalizedPath = normalizeSeparators(path);
  const normalizedTarget = normalizeSeparators(targetRelativePath).replace(/^\/+|\/+$/g, '');

  if (normalizedTarget === '') {
    return normalizedPath;
  }

  if (normalizedPath === normalizedTarget) {
    return filename(normalizedPath);
  }

  const targetPrefix = `${normalizedTarget}/`;
  if (normalizedPath.startsWith(targetPrefix)) {
    return normalizedPath.slice(targetPrefix.length);
  }

  const embeddedTargetPrefix = `/${targetPrefix}`;
  const embeddedTargetIndex = normalizedPath.toLowerCase().lastIndexOf(embeddedTargetPrefix.toLowerCase());
  if (embeddedTargetIndex >= 0) {
    return normalizedPath.slice(embeddedTargetIndex + embeddedTargetPrefix.length);
  }

  return filename(normalizedPath);
};

const operationDescription = (operation: ReShadeOperation, targetRelativePath: string) => {
  const targetPath = displayPath(operation.targetPath, targetRelativePath);

  if (operation.type === 'restore' || operation.type === 'move') {
    return `${displayPath(operation.sourcePath ?? '', targetRelativePath)} -> ${targetPath}`;
  }
  return targetPath;
};

const PreviewGroup = ({ items, title, tone }: PreviewGroupProps) => {
  if (items.length === 0) {
    return null;
  }
  return (
    <section
      className={
        tone === undefined
          ? 'reshade-preview-group'
          : `reshade-preview-group reshade-preview-group-${tone}`
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

const pathImpactAction = (impact: PathImpact) => {
  if (impact.action === 'replace' && !impact.exists) {
    return 'add';
  }
  if (impact.action === 'create') {
    return 'add';
  }
  if (impact.action === 'update search paths') {
    return 'update';
  }
  return impact.action;
};

const pathImpactLabel = (targetRelativePath: string) => (impact: PathImpact) =>
  `${pathImpactAction(impact)}: ${displayPath(impact.path, targetRelativePath)}`;

const filesToAdd = (operations: ReShadeOperation[]) =>
  operations.filter((operation) => operation.type === 'copy' || operation.type === 'adopt');

const filesToRemove = (operations: ReShadeOperation[]) =>
  operations.filter(
    (operation) =>
      operation.type === 'delete' || operation.type === 'move' || operation.type === 'restore',
  );

const chainItems = (chainTarget: ReShadeChainTarget | null) => {
  if (chainTarget === null) {
    return [];
  }
  return [
    `${chainTarget.PrimaryOwner} owns ${chainTarget.PrimaryProxyFilename}`,
    ...(chainTarget.OptiScaler !== null
      ? [`OptiScaler: ${chainTarget.OptiScaler.ProxyFilename}`]
      : []),
    ...(chainTarget.ReShade !== null
      ? [`ReShade: ${chainTarget.ReShade.ActiveRuntimeFilename}`]
      : []),
  ];
};

const impactsByRole = (impacts: PathImpact[], roles: string[]) =>
  impacts.filter((impact) => roles.includes(impact.role));

export const ReShadePreview = ({ chainTarget, preview }: ReShadePreviewProps) => {
  const visibleImpacts = preview.pathImpacts.filter((impact) => !impact.preservationOnly);
  const runtimeImpacts = impactsByRole(visibleImpacts, ['runtime']);
  const configurationImpacts = impactsByRole(visibleImpacts, ['configuration', 'preset']);
  const contentImpacts = impactsByRole(visibleImpacts, ['effects', 'textures', 'addons']);
  const backupImpacts = impactsByRole(visibleImpacts, ['backup']);
  const formatPathImpact = pathImpactLabel(preview.request.targetRelativePath);

  return (
    <div className="reshade-preview">
      <PreviewGroup items={preview.conflicts} title="Blocking conflicts" tone="danger" />
      <PreviewGroup
        items={preview.drift.map(
          (drift) => `${drift.relativePath}${drift.missing ? ' is missing' : ' has changed'}`,
        )}
        title="Managed file drift"
        tone="warning"
      />
      <PreviewGroup
        items={preview.userContentDrift.map(
          (drift) => `${drift.path}${drift.missing ? ' is missing' : ' has changed'}`,
        )}
        title="User content drift"
        tone="warning"
      />
      <PreviewGroup items={preview.warnings} title="Warnings" tone="warning" />
      <PreviewGroup items={runtimeImpacts.map(formatPathImpact)} title="Runtime files" />
      <PreviewGroup
        items={configurationImpacts.map(formatPathImpact)}
        title="Configuration and presets"
      />
      <PreviewGroup
        items={contentImpacts.map(formatPathImpact)}
        title="Effects, textures, and add-ons"
      />
      <PreviewGroup items={backupImpacts.map(formatPathImpact)} title="Backups" tone="warning" />
      <PreviewGroup
        items={filesToAdd(preview.operations).map((operation) => (
          <>
            <span className="reshade-preview-symbol">+</span>
            {operationDescription(operation, preview.request.targetRelativePath)}
          </>
        ))}
        title="Files to add or update"
        tone="success"
      />
      <PreviewGroup
        items={filesToRemove(preview.operations).map((operation) => (
          <>
            <span className="reshade-preview-symbol">-</span>
            {operationDescription(operation, preview.request.targetRelativePath)}
          </>
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
