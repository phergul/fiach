import { describe, expect, it } from 'vitest';

import {
  resolveDeploymentActionLabel,
  resolveDeploymentActionTone,
  resolveDeploymentSummaryTone,
} from './deploymentLabels';

describe('deploymentLabels', () => {
  it('labels incremental statuses and actions', () => {
    expect(resolveDeploymentActionLabel('drifted', 'require_decision')).toBe('Drifted');
    expect(resolveDeploymentActionLabel('external', 'noop')).toBe('External');
    expect(resolveDeploymentActionLabel('unchanged', 'noop')).toBe('Unchanged');
    expect(resolveDeploymentActionLabel('added', 'create')).toBe('Create');
    expect(resolveDeploymentActionLabel('added', 'require_decision')).toBe('Decision required');
  });

  it('maps incremental tones', () => {
    expect(resolveDeploymentActionTone('drifted', 'require_decision')).toBe('warning');
    expect(resolveDeploymentActionTone('external', 'noop')).toBe('info');
    expect(resolveDeploymentActionTone('skipped', 'noop')).toBe('warning');
    expect(resolveDeploymentActionTone('unchanged', 'noop')).toBe('default');
    expect(resolveDeploymentSummaryTone('drifted')).toBe('warning');
    expect(resolveDeploymentSummaryTone('external')).toBe('info');
    expect(resolveDeploymentSummaryTone('skipped')).toBe('warning');
  });
});
