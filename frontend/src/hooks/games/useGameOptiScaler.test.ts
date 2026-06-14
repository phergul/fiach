import { describe, expect, it } from 'vitest';

import type {
  OptiScalerCandidate,
  OptiScalerRecoveryState,
  OptiScalerTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';

import { getOptiScalerAggregateStatus } from './useGameOptiScaler';

const candidate = {
  architecture: 'x64',
  evidence: [],
  executableName: 'Game.exe',
  executableRelativePath: 'Game.exe',
  hasOptiScaler: false,
  hasReShade: false,
  managed: false,
  targetRelativePath: '.',
} as OptiScalerCandidate;

const target = {
  ReleaseTag: 'v1',
  Status: 'managed',
} as OptiScalerTarget;

describe('getOptiScalerAggregateStatus', () => {
  it('uses the required priority order', () => {
    expect(getOptiScalerAggregateStatus([candidate], [{ ...target, Status: 'drifted' }], null, 'v2', null)).toBe('drift');
    expect(getOptiScalerAggregateStatus([candidate], [target], null, 'v2', null)).toBe('update');
    expect(getOptiScalerAggregateStatus([candidate], [target], null, 'v1', null)).toBe('managed');
    expect(getOptiScalerAggregateStatus([candidate], [], null, 'v1', null)).toBe('unmanaged');
    expect(getOptiScalerAggregateStatus([], [], null, 'v1', null)).toBe('not_detected');
  });

  it('prioritizes a global recovery journal over target state', () => {
    const recovery = { required: true } as OptiScalerRecoveryState;
    expect(getOptiScalerAggregateStatus(
      [candidate],
      [{ ...target, Status: 'drifted' }],
      recovery,
      'v2',
      null,
    )).toBe('recovery');
  });
});
