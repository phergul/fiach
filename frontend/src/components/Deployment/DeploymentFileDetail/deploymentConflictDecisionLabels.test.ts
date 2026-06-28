import { describe, expect, it } from 'vitest';

import {
  resolveConflictActionLabel,
  shouldShowConflictDecisionPanel,
} from './deploymentConflictDecisionLabels';

type TestDetail = Parameters<typeof shouldShowConflictDecisionPanel>[0];

const buildDetail = (overrides: Partial<TestDetail> = {}): TestDetail => {
  return {
    RelativePath: 'Shared/plugin.txt',
    ConflictCategory: 'expected_overwrite',
    ConflictAvailableActions: [],
    SavedConflictRuleModID: null,
    SavedConflictRuleModName: '',
    ProfileModsURL: '',
    WriterStack: [
      {
        Order: 1,
        SourceKind: 'mod',
        ModID: 1,
        ModName: 'Alpha',
        LoadOrder: 0,
        IsWinner: false,
        WouldWrite: true,
        SourceID: 'mod:1',
      },
      {
        Order: 2,
        SourceKind: 'mod',
        ModID: 2,
        ModName: 'Beta',
        LoadOrder: 1,
        IsWinner: true,
        WouldWrite: false,
        SourceID: 'mod:2',
      },
    ],
    ...overrides,
  } as TestDetail;
};

describe('deploymentConflictDecisionLabels', () => {
  it('shows the panel for multi-writer expected overwrite conflicts', () => {
    expect(shouldShowConflictDecisionPanel(buildDetail())).toBe(true);
  });

  it('hides the panel for single-writer paths without saved rules', () => {
    expect(
      shouldShowConflictDecisionPanel(
        buildDetail({
          WriterStack: [
            {
              Order: 1,
              SourceKind: 'mod',
              ModID: 1,
              ModName: 'Alpha',
              LoadOrder: 0,
              DisplayLoadOrder: 1,
              IsWinner: true,
              WouldWrite: false,
              SourceID: 'mod:1',
            },
          ],
        }),
      ),
    ).toBe(false);
  });

  it('labels per-mod winner actions from the writer stack', () => {
    const detail = buildDetail();
    expect(resolveConflictActionLabel('set_per_file_winner:2', detail)).toBe(
      'Use Beta for this file',
    );
    expect(resolveConflictActionLabel('clear_conflict_rule', detail)).toBe('Clear per-file rule');
  });
});
