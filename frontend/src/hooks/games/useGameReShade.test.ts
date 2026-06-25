import { describe, expect, it } from 'vitest';

import type {
  ReShadeDiscoveryResult,
  ReShadeTarget,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { Architecture } from '@bindings/github.com/phergul/fiach/internal/reshade/models';

import { getReShadeAggregateStatus } from './useGameReShade';

const target = {
  Status: 'managed',
} as ReShadeTarget;

const discovery = (isReShade: boolean): ReShadeDiscoveryResult => ({
  candidates: [
    {
      apiOptions: [],
      architecture: Architecture.ArchitectureX64,
      conflicts: [],
      executableRelativePath: 'Game.exe',
      proxyEvidence: [
        {
          exists: true,
          filename: 'dxgi.dll',
          isReShade,
        },
      ],
      targetRelativePath: '.',
    },
  ],
  warnings: [],
});

describe('getReShadeAggregateStatus', () => {
  it('only reports unmanaged when ReShade files are positively detected', () => {
    expect(getReShadeAggregateStatus(discovery(true), [], null, null)).toBe('unmanaged');
    expect(getReShadeAggregateStatus(discovery(false), [], null, null)).toBe('not_detected');
  });

  it('prioritizes managed targets over unmanaged discovery', () => {
    expect(getReShadeAggregateStatus(discovery(true), [target], null, null)).toBe('managed');
  });
});
