import { useState } from 'react';

import { ChevronDown, ChevronRight } from 'lucide-react';

import {
  OperationType,
  PlanIssueKind,
  PlanIssueSeverity,
  type Operation,
  type OperationPlan,
  type PlanIssue,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import './GameApplyReview.scss';

type ReviewTone = 'blocked' | 'warning' | 'add' | 'replace' | 'folder';

interface ReviewRow {
  details?: string[];
  id: string;
  meta: string[];
  title: string;
  tone: ReviewTone;
}

interface ReviewGroup {
  defaultCollapsed: boolean;
  id: string;
  count: number;
  emptyMessage: string;
  rows: ReviewRow[];
  title: string;
}

interface GameApplyReviewProps {
  gameInstallPath: string;
  gameName: string;
  plan: OperationPlan;
}

const normalizePath = (path: string) => {
  return path.trim().replace(/\\/g, '/').replace(/\/+$/g, '');
};

const baseName = (path: string) => {
  const normalizedPath = normalizePath(path);
  const pathParts = normalizedPath.split('/').filter(Boolean);

  return pathParts[pathParts.length - 1] ?? '';
};

const formatTargetPath = (targetPath: string, gameInstallPath: string, gameName: string) => {
  const normalizedTargetPath = normalizePath(targetPath);
  const normalizedInstallPath = normalizePath(gameInstallPath);
  const installFolderName = baseName(gameInstallPath) || gameName;

  if (normalizedInstallPath === '') {
    return normalizedTargetPath;
  }

  const targetPathLower = normalizedTargetPath.toLowerCase();
  const installPathLower = normalizedInstallPath.toLowerCase();

  if (targetPathLower === installPathLower) {
    return installFolderName;
  }
  if (targetPathLower.startsWith(`${installPathLower}/`)) {
    return `${installFolderName}/${normalizedTargetPath.slice(normalizedInstallPath.length + 1)}`;
  }

  return normalizedTargetPath;
};

const formatIssueMeta = (issue: PlanIssue) => {
  const meta: string[] = [];

  if (issue.Mod !== null && issue.Mod.ModName.trim() !== '') {
    meta.push(issue.Mod.ModName);
  }
  if (issue.TargetPath !== null && issue.TargetPath.trim() !== '') {
    meta.push(`Target: ${issue.TargetPath}`);
  }
  if (issue.SourcePath !== null && issue.SourcePath.trim() !== '') {
    meta.push(`Source: ${issue.SourcePath}`);
  }

  return meta;
};

const formatOperationMeta = (operation: Operation) => {
  const meta: string[] = [];

  if (operation.Mod.ModName.trim() !== '') {
    meta.push(operation.Mod.ModName);
  }
  if (operation.Conflict) {
    meta.push('Conflicting target');
  }
  if (operation.BackupPath !== null && operation.BackupPath.trim() !== '') {
    meta.push('Existing file will be backed up');
  }

  return meta;
};

const formatOperationType = (type: OperationType) => {
  switch (type) {
    case OperationType.OperationTypeCopy:
      return 'Copy';
    case OperationType.OperationTypeReplace:
      return 'Replace';
    case OperationType.OperationTypeCreateDirectory:
      return 'Create folder';
    default:
      return 'Operation';
  }
};

const pluralizeMods = (count: number) => {
  return `${count} ${count === 1 ? 'mod' : 'mods'} conflict`;
};

const resolveConflictOperations = (issue: PlanIssue, operations: Operation[]) => {
  const seenIndexes = new Set<number>();
  const conflictOperations: Operation[] = [];

  for (const index of issue.ConflictingOperationIndexes) {
    if (seenIndexes.has(index) || index < 0 || index >= operations.length) {
      continue;
    }

    seenIndexes.add(index);
    conflictOperations.push(operations[index]);
  }

  return conflictOperations.sort((left, right) => {
    const modNameCompare = left.Mod.ModName.localeCompare(right.Mod.ModName);
    if (modNameCompare !== 0) {
      return modNameCompare;
    }

    return formatOperationType(left.Type).localeCompare(formatOperationType(right.Type));
  });
};

const formatConflictOperationDetail = (operation: Operation) => {
  const detail = [operation.Mod.ModName.trim() || 'Unknown mod', formatOperationType(operation.Type)];

  if (operation.SourcePath !== null && operation.SourcePath.trim() !== '') {
    detail.push(`Source: ${operation.SourcePath}`);
  }

  return detail.join(' · ');
};

const issueRows = (issues: PlanIssue[], severity: PlanIssueSeverity, tone: ReviewTone) => {
  return issues
    .filter((issue) => issue.Severity === severity && issue.Kind !== PlanIssueKind.PlanIssueTargetPathConflict)
    .map((issue, index) => ({
      id: `${severity}-${issue.Kind}-${index}`,
      meta: formatIssueMeta(issue),
      title: issue.Message,
      tone,
    }));
};

const conflictRows = (issues: PlanIssue[], operations: Operation[], gameInstallPath: string, gameName: string) => {
  return issues
    .filter(
      (issue) =>
        issue.Severity === PlanIssueSeverity.PlanIssueSeverityError &&
        issue.Kind === PlanIssueKind.PlanIssueTargetPathConflict,
    )
    .map((issue, index) => {
      const conflictOperations = resolveConflictOperations(issue, operations);
      const title =
        issue.TargetPath !== null && issue.TargetPath.trim() !== ''
          ? formatTargetPath(issue.TargetPath, gameInstallPath, gameName)
          : issue.Message;

      return {
        details: conflictOperations.map(formatConflictOperationDetail),
        id: `conflict-${issue.TargetPath ?? index}-${index}`,
        meta: conflictOperations.length > 0 ? [pluralizeMods(conflictOperations.length)] : formatIssueMeta(issue),
        title,
        tone: 'blocked' as ReviewTone,
      };
    });
};

const operationRows = (
  operations: Operation[],
  type: OperationType,
  tone: ReviewTone,
  gameInstallPath: string,
  gameName: string,
) => {
  return operations
    .filter((operation) => operation.Type === type)
    .map((operation, index) => ({
      id: `${type}-${operation.TargetPath}-${index}`,
      meta: formatOperationMeta(operation),
      title: formatTargetPath(operation.TargetPath, gameInstallPath, gameName),
      tone,
    }));
};

const buildGroups = (plan: OperationPlan, gameInstallPath: string, gameName: string): ReviewGroup[] => {
  const blockingRows = [
    ...conflictRows(plan.Issues, plan.Operations, gameInstallPath, gameName),
    ...issueRows(plan.Issues, PlanIssueSeverity.PlanIssueSeverityError, 'blocked'),
  ];
  const warningRows = issueRows(plan.Issues, PlanIssueSeverity.PlanIssueSeverityWarning, 'warning');
  const addRows = operationRows(plan.Operations, OperationType.OperationTypeCopy, 'add', gameInstallPath, gameName);
  const replaceRows = operationRows(plan.Operations, OperationType.OperationTypeReplace, 'replace', gameInstallPath, gameName);
  const folderRows = operationRows(
    plan.Operations,
    OperationType.OperationTypeCreateDirectory,
    'folder',
    gameInstallPath,
    gameName,
  );

  const issueGroups: ReviewGroup[] = [
    {
      count: blockingRows.length,
      defaultCollapsed: false,
      emptyMessage: 'No blocking issues.',
      id: 'blocking',
      rows: blockingRows,
      title: 'Blocking issues',
    },
    {
      count: warningRows.length,
      defaultCollapsed: false,
      emptyMessage: 'No warnings.',
      id: 'warnings',
      rows: warningRows,
      title: 'Warnings',
    },
  ].filter((group) => group.count > 0);

  const operationGroups: ReviewGroup[] = [
    {
      count: addRows.length,
      defaultCollapsed: addRows.length === 0,
      emptyMessage: 'No files to add.',
      id: 'files-add',
      rows: addRows,
      title: 'Files to add',
    },
    {
      count: replaceRows.length,
      defaultCollapsed: replaceRows.length === 0,
      emptyMessage: 'No files to replace.',
      id: 'files-replace',
      rows: replaceRows,
      title: 'Files to replace',
    },
    {
      count: folderRows.length,
      defaultCollapsed: folderRows.length === 0,
      emptyMessage: 'No folders to create.',
      id: 'folders-create',
      rows: folderRows,
      title: 'Folders to create',
    },
  ];

  return [...issueGroups, ...operationGroups];
};

const toneLabel: Record<ReviewTone, string> = {
  add: 'Add',
  blocked: 'Blocked',
  folder: 'Create folder',
  replace: 'Replace',
  warning: 'Warning',
};

export const GameApplyReview = ({ gameInstallPath, gameName, plan }: GameApplyReviewProps) => {
  const groups = buildGroups(plan, gameInstallPath, gameName);
  const [collapsedGroups, setCollapsedGroups] = useState<Record<string, boolean>>({});

  const toggleGroup = (groupID: string) => {
    setCollapsedGroups((currentGroups) => ({
      ...currentGroups,
      [groupID]: !currentGroups[groupID],
    }));
  };

  return (
    <section className="game-apply-review" aria-label="Operation review">
      {groups.map((group) => {
        const isCollapsed = collapsedGroups[group.id] ?? group.defaultCollapsed;
        const hasRows = group.rows.length > 0;

        return (
          <section className="game-apply-review-group" key={group.id} aria-label={group.title}>
            <button
              className="game-apply-review-group-header"
              onClick={() => toggleGroup(group.id)}
              type="button"
              aria-expanded={!isCollapsed}
            >
              {isCollapsed ? (
                <ChevronRight className="game-apply-review-group-icon" aria-hidden="true" />
              ) : (
                <ChevronDown className="game-apply-review-group-icon" aria-hidden="true" />
              )}
              <span className="game-apply-review-group-title">{group.title}</span>
              <span className="game-apply-review-group-count">{group.count}</span>
            </button>

            {!isCollapsed && (
              <div className="game-apply-review-rows">
                {!hasRows && <p className="game-apply-review-empty">{group.emptyMessage}</p>}

                {group.rows.map((row) => (
                  <article className="game-apply-review-row" key={row.id}>
                    <div className="game-apply-review-row-copy">
                      <p className="game-apply-review-row-title">{row.title}</p>
                      {row.meta.length > 0 && (
                        <p className="game-apply-review-row-meta">{row.meta.join(' · ')}</p>
                      )}
                      {row.details !== undefined && row.details.length > 0 && (
                        <ul className="game-apply-review-row-details">
                          {row.details.map((detail, detailIndex) => (
                            <li className="game-apply-review-row-detail" key={`${detail}-${detailIndex}`}>
                              {detail}
                            </li>
                          ))}
                        </ul>
                      )}
                    </div>
                    <span className={`game-apply-review-chip game-apply-review-chip-${row.tone}`}>
                      {toneLabel[row.tone]}
                    </span>
                  </article>
                ))}
              </div>
            )}
          </section>
        );
      })}
    </section>
  );
};
