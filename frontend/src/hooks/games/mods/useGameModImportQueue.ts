import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import type {
  ImportSourceRef,
  StrategyType,
} from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import {
  ImportMod,
  PreValidateImport,
  ResolveImportSourceDuplicates,
} from '@bindings/github.com/phergul/fiach/internal/services/modservice';
import { ResolveGameModStoragePath } from '@bindings/github.com/phergul/fiach/internal/services/settingsservice';
import { useToast } from '@components/Common/Toast/Toast';
import type { ModTagSelection } from '@components/Games/Details/Mods/ModTags/ModTagEditor/ModTagEditor';
import {
  getArchiveImportName,
  getErrorMessage,
  getFolderImportName,
  getImportSourceLabel,
  inferImportSourceType,
  openArchives,
  openDirectories,
} from '@utils';

export type ImportQueueItemStatus =
  | 'pending'
  | 'reviewing'
  | 'importing'
  | 'imported'
  | 'failed'
  | 'skipped';

export interface ImportQueueItem {
  canonicalPath: string;
  error?: string;
  id: string;
  importedModName?: string;
  initialName: string;
  sourceLabel: string;
  sourcePath: string;
  sourceType: ModSourceType;
  status: ImportQueueItemStatus;
  statusMessage?: string;
  suggestedStrategy: StrategyType | null;
  targetPath: string;
}

export type ImportQueueViewMode = 'idle' | 'wizard' | 'queue' | 'summary';

interface ImportWizardSubmitInput {
  name: string;
  strategyType: StrategyType;
  targetRelativePath: string;
  tags: ModTagSelection[];
}

interface LastImportSettings {
  strategyType: StrategyType;
  targetRelativePath: string;
}

interface UseGameModImportQueueInput {
  gameID: number | null;
  refreshMods: () => Promise<unknown>;
}

const createQueueItemID = () =>
  globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(16).slice(2)}`;

const isActionableQueueItem = (item: ImportQueueItem) =>
  item.status === 'pending' ||
  item.status === 'reviewing' ||
  item.status === 'importing' ||
  item.status === 'imported' ||
  item.status === 'failed';

const getImportNameForSource = (sourceType: ModSourceType, sourcePath: string) =>
  sourceType === ModSourceType.ModSourceTypeArchive
    ? getArchiveImportName(sourcePath)
    : getFolderImportName(sourcePath);

export const useGameModImportQueue = ({ gameID, refreshMods }: UseGameModImportQueueInput) => {
  const { addErrorToast, addToast } = useToast();
  const [items, setItems] = useState<ImportQueueItem[]>([]);
  const [viewMode, setViewMode] = useState<ImportQueueViewMode>('idle');
  const [currentItemID, setCurrentItemID] = useState<string | null>(null);
  const [showQueueChrome, setShowQueueChrome] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const [isEnqueueing, setIsEnqueueing] = useState(false);
  const [isImportMenuOpen, setIsImportMenuOpen] = useState(false);
  const [reusePreviousImportSettings, setReusePreviousImportSettings] = useState(true);
  const [importAnotherAfterComplete, setImportAnotherAfterComplete] = useState(false);
  const [lastImportSettings, setLastImportSettings] = useState<LastImportSettings | null>(null);
  const [pendingImportAnotherSourceType, setPendingImportAnotherSourceType] =
    useState<ModSourceType | null>(null);
  const importAnotherQueueSnapshotRef = useRef<ImportQueueItem[] | null>(null);
  const importAnotherPickerActiveRef = useRef(false);

  const currentItem = useMemo(
    () =>
      currentItemID === null ? null : (items.find((item) => item.id === currentItemID) ?? null),
    [currentItemID, items],
  );

  const actionableItems = useMemo(() => items.filter(isActionableQueueItem), [items]);

  const queuePosition = useMemo(() => {
    if (currentItem === null) {
      return null;
    }

    const currentIndex = actionableItems.findIndex((item) => item.id === currentItem.id);
    if (currentIndex === -1) {
      return null;
    }

    return {
      current: currentIndex + 1,
      total: actionableItems.length,
    };
  }, [actionableItems, currentItem]);

  const summaryCounts = useMemo(() => {
    return items.reduce(
      (counts, item) => {
        if (item.status === 'imported') {
          counts.imported += 1;
        } else if (item.status === 'failed') {
          counts.failed += 1;
        } else if (item.status === 'skipped') {
          counts.skipped += 1;
        }
        return counts;
      },
      { failed: 0, imported: 0, skipped: 0 },
    );
  }, [items]);

  const isBusy = isImporting || isEnqueueing;

  const openWizardForItem = useCallback((itemID: string) => {
    setCurrentItemID(itemID);
    setShowQueueChrome(true);
    setViewMode('wizard');
    setImportError(null);
    setItems((currentItems) =>
      currentItems.map((item) =>
        item.id === itemID
          ? {
              ...item,
              error: undefined,
              status: 'reviewing' as const,
            }
          : item.status === 'reviewing'
            ? { ...item, status: 'pending' as const }
            : item,
      ),
    );
  }, []);

  const showPostImportState = useCallback((remainingItems: ImportQueueItem[]) => {
    setImportAnotherAfterComplete(false);
    setPendingImportAnotherSourceType(null);

    const actionable = remainingItems.filter(isActionableQueueItem);
    if (actionable.length === 0) {
      setViewMode('idle');
      setCurrentItemID(null);
      setShowQueueChrome(false);
      setItems([]);
      return;
    }

    setItems(remainingItems);
    setCurrentItemID(null);
    setShowQueueChrome(false);

    if (actionable.length > 1) {
      setViewMode('summary');
      return;
    }

    setViewMode('idle');
    setItems([]);
  }, []);

  const finishQueueFlow = useCallback(
    (nextItems: ImportQueueItem[], shouldImportAnother = false) => {
      const hasPending = nextItems.some((item) => item.status === 'pending');
      if (hasPending) {
        return;
      }

      const hasActionable = nextItems.some(isActionableQueueItem);
      if (!hasActionable) {
        setViewMode('idle');
        setCurrentItemID(null);
        setShowQueueChrome(false);
        setItems([]);
        setImportAnotherAfterComplete(false);
        setPendingImportAnotherSourceType(null);
        return;
      }

      if (shouldImportAnother) {
        const lastImported = [...nextItems].reverse().find((item) => item.status === 'imported');
        importAnotherQueueSnapshotRef.current = nextItems;
        setViewMode('idle');
        setCurrentItemID(null);
        setShowQueueChrome(false);
        setItems(nextItems);
        setPendingImportAnotherSourceType(
          lastImported?.sourceType ?? ModSourceType.ModSourceTypeFolder,
        );
        return;
      }

      const actionable = nextItems.filter(isActionableQueueItem);
      if (actionable.length > 1) {
        setViewMode('summary');
        setCurrentItemID(null);
        setShowQueueChrome(false);
        setImportAnotherAfterComplete(false);
        setPendingImportAnotherSourceType(null);
        return;
      }

      setViewMode('idle');
      setCurrentItemID(null);
      setShowQueueChrome(false);
      setItems([]);
      setImportAnotherAfterComplete(false);
      setPendingImportAnotherSourceType(null);
    },
    [],
  );

  const advanceAfterSuccess = useCallback(
    (nextItems: ImportQueueItem[], shouldImportAnother = false) => {
      const nextPending = nextItems.find((item) => item.status === 'pending');
      if (nextPending !== undefined) {
        setItems(nextItems);
        openWizardForItem(nextPending.id);
        return;
      }

      setItems(nextItems);
      finishQueueFlow(nextItems, shouldImportAnother);
    },
    [finishQueueFlow, openWizardForItem],
  );

  const enqueueSources = useCallback(
    async (
      sources: ImportSourceRef[],
      options?: { append?: boolean; baseItems?: ImportQueueItem[] },
    ) => {
      if (gameID === null || isBusy) {
        return;
      }

      if (sources.length === 0) {
        return;
      }

      setIsEnqueueing(true);
      setIsImportMenuOpen(false);
      setImportError(null);
      if (!options?.append) {
        setLastImportSettings(null);
        setReusePreviousImportSettings(true);
        setImportAnotherAfterComplete(false);
      }

      const existingItems = options?.append ? (options.baseItems ?? items) : [];

      try {
        const duplicateResult = await ResolveImportSourceDuplicates({
          GameID: gameID,
          Sources: sources,
        });

        const existingCanonicalPaths = new Set(
          existingItems
            .filter((item) => item.canonicalPath !== '')
            .map((item) => item.canonicalPath),
        );

        const nextItems: ImportQueueItem[] = options?.append ? [...existingItems] : [];
        const pendingItems: ImportQueueItem[] = [];

        for (const status of duplicateResult.Items) {
          if (status.Error !== null && status.Error !== undefined && status.Error.trim() !== '') {
            nextItems.push({
              canonicalPath: status.CanonicalPath,
              id: createQueueItemID(),
              initialName: getImportNameForSource(status.SourceType, status.SourcePath),
              sourceLabel: getImportSourceLabel(
                status.SourceType === ModSourceType.ModSourceTypeArchive ? 'archive' : 'folder',
              ),
              sourcePath: status.SourcePath,
              sourceType: status.SourceType,
              status: 'skipped',
              statusMessage: status.Error,
              suggestedStrategy: null,
              targetPath: '',
            });
            continue;
          }

          if (status.IsDuplicate) {
            const duplicateName = status.ExistingModName ?? 'an existing mod';
            nextItems.push({
              canonicalPath: status.CanonicalPath,
              id: createQueueItemID(),
              initialName: getImportNameForSource(status.SourceType, status.SourcePath),
              sourceLabel: getImportSourceLabel(
                status.SourceType === ModSourceType.ModSourceTypeArchive ? 'archive' : 'folder',
              ),
              sourcePath: status.SourcePath,
              sourceType: status.SourceType,
              status: 'skipped',
              statusMessage: `Already imported as ${duplicateName}`,
              suggestedStrategy: null,
              targetPath: '',
            });
            continue;
          }

          if (status.CanonicalPath !== '' && existingCanonicalPaths.has(status.CanonicalPath)) {
            nextItems.push({
              canonicalPath: status.CanonicalPath,
              id: createQueueItemID(),
              initialName: getImportNameForSource(status.SourceType, status.SourcePath),
              sourceLabel: getImportSourceLabel(
                status.SourceType === ModSourceType.ModSourceTypeArchive ? 'archive' : 'folder',
              ),
              sourcePath: status.SourcePath,
              sourceType: status.SourceType,
              status: 'skipped',
              statusMessage: 'Already in queue',
              suggestedStrategy: null,
              targetPath: '',
            });
            continue;
          }

          const validation = await PreValidateImport({
            SourceType: status.SourceType,
            SourcePath: status.SourcePath,
          });
          const targetPath = await ResolveGameModStoragePath(gameID);
          const queueItem: ImportQueueItem = {
            canonicalPath: status.CanonicalPath,
            id: createQueueItemID(),
            initialName: getImportNameForSource(status.SourceType, status.SourcePath),
            sourceLabel: getImportSourceLabel(
              status.SourceType === ModSourceType.ModSourceTypeArchive ? 'archive' : 'folder',
            ),
            sourcePath: status.SourcePath,
            sourceType: status.SourceType,
            status: 'pending',
            suggestedStrategy: validation.SuggestedStrategy,
            targetPath,
          };

          if (status.CanonicalPath !== '') {
            existingCanonicalPaths.add(status.CanonicalPath);
          }

          nextItems.push(queueItem);
          pendingItems.push(queueItem);
        }

        if (pendingItems.length === 0) {
          const showSummary = nextItems.filter(isActionableQueueItem).length > 1;
          setItems(nextItems);
          if (showSummary) {
            setViewMode('summary');
          } else {
            setViewMode('idle');
            setItems([]);
          }
          setCurrentItemID(null);
          setShowQueueChrome(false);
          return;
        }

        setItems(nextItems);
        openWizardForItem(pendingItems[0].id);
      } catch (error) {
        addErrorToast(error);
      } finally {
        setIsEnqueueing(false);
      }
    },
    [addErrorToast, gameID, isBusy, items, openWizardForItem],
  );

  useEffect(() => {
    if (
      pendingImportAnotherSourceType === null ||
      gameID === null ||
      isImporting ||
      isEnqueueing ||
      importAnotherPickerActiveRef.current
    ) {
      return;
    }

    const sourceType = pendingImportAnotherSourceType;
    const queueSnapshot = importAnotherQueueSnapshotRef.current ?? [];
    importAnotherPickerActiveRef.current = true;

    const runImportAnother = async () => {
      try {
        const selectedPaths =
          sourceType === ModSourceType.ModSourceTypeArchive
            ? await openArchives({
                buttonText: 'Add to Import Queue',
                title: 'Select mod archives',
              })
            : await openDirectories({
                buttonText: 'Add to Import Queue',
                title: 'Select mod folders',
              });

        if (selectedPaths === null || selectedPaths.length === 0) {
          showPostImportState(queueSnapshot);
          return;
        }

        await enqueueSources(
          selectedPaths.map((sourcePath) => ({
            SourcePath: sourcePath,
            SourceType: sourceType,
          })),
          { append: true, baseItems: queueSnapshot },
        );
      } catch (error) {
        addErrorToast(error);
        showPostImportState(queueSnapshot);
      } finally {
        setImportAnotherAfterComplete(false);
        setPendingImportAnotherSourceType(null);
        importAnotherQueueSnapshotRef.current = null;
        importAnotherPickerActiveRef.current = false;
      }
    };

    void runImportAnother();
  }, [
    addErrorToast,
    enqueueSources,
    gameID,
    isEnqueueing,
    isImporting,
    pendingImportAnotherSourceType,
    showPostImportState,
  ]);

  const startFolderImportFlow = useCallback(async () => {
    if (gameID === null || isBusy) {
      return;
    }

    try {
      const selectedPaths = await openDirectories({
        buttonText: 'Add to Import Queue',
        title: 'Select mod folders',
      });
      if (selectedPaths === null || selectedPaths.length === 0) {
        return;
      }

      await enqueueSources(
        selectedPaths.map((sourcePath) => ({
          SourcePath: sourcePath,
          SourceType: ModSourceType.ModSourceTypeFolder,
        })),
        { append: viewMode === 'queue' || viewMode === 'wizard' },
      );
    } catch (error) {
      addErrorToast(error);
    }
  }, [addErrorToast, enqueueSources, gameID, isBusy, viewMode]);

  const startArchiveImportFlow = useCallback(async () => {
    if (gameID === null || isBusy) {
      return;
    }

    try {
      const selectedPaths = await openArchives({
        buttonText: 'Add to Import Queue',
        title: 'Select mod archives',
      });
      if (selectedPaths === null || selectedPaths.length === 0) {
        return;
      }

      await enqueueSources(
        selectedPaths.map((sourcePath) => ({
          SourcePath: sourcePath,
          SourceType: ModSourceType.ModSourceTypeArchive,
        })),
        { append: viewMode === 'queue' || viewMode === 'wizard' },
      );
    } catch (error) {
      addErrorToast(error);
    }
  }, [addErrorToast, enqueueSources, gameID, isBusy, viewMode]);

  const handleDroppedFiles = useCallback(
    async (paths: string[]) => {
      if (gameID === null || isBusy || paths.length === 0) {
        return;
      }

      await enqueueSources(
        paths.map((sourcePath) => ({
          SourcePath: sourcePath,
          SourceType: inferImportSourceType(sourcePath),
        })),
        { append: viewMode !== 'idle' },
      );
    },
    [enqueueSources, gameID, isBusy, viewMode],
  );

  const reviewItem = useCallback(
    (itemID: string) => {
      if (isBusy) {
        return;
      }

      openWizardForItem(itemID);
    },
    [isBusy, openWizardForItem],
  );

  const skipItem = useCallback(
    (itemID: string) => {
      if (isBusy) {
        return;
      }

      setItems((currentItems) => {
        const nextItems = currentItems.map((item) =>
          item.id === itemID && item.status === 'pending'
            ? { ...item, status: 'skipped' as const, statusMessage: 'Skipped' }
            : item,
        );
        finishQueueFlow(nextItems);
        return nextItems;
      });
    },
    [finishQueueFlow, isBusy],
  );

  const removeItem = useCallback(
    (itemID: string) => {
      if (isBusy) {
        return;
      }

      setItems((currentItems) => {
        const nextItems = currentItems.filter((item) => item.id !== itemID);
        finishQueueFlow(nextItems);
        return nextItems;
      });
    },
    [finishQueueFlow, isBusy],
  );

  const closeImportReview = useCallback(() => {
    if (isImporting) {
      return;
    }

    setImportError(null);

    if (currentItemID !== null) {
      setItems((currentItems) =>
        currentItems.map((item) =>
          item.id === currentItemID && item.status === 'reviewing'
            ? { ...item, status: 'pending' as const }
            : item,
        ),
      );
    }

    if (showQueueChrome) {
      setViewMode('queue');
      setCurrentItemID(null);
      return;
    }

    setViewMode('idle');
    setCurrentItemID(null);
    setItems([]);
    setShowQueueChrome(false);
    setImportError(null);
    setImportAnotherAfterComplete(false);
    setPendingImportAnotherSourceType(null);
  }, [currentItemID, isImporting, showQueueChrome]);

  const importCurrentItem = useCallback(
    async ({ name, strategyType, targetRelativePath, tags }: ImportWizardSubmitInput) => {
      if (gameID === null || currentItem === null || isImporting) {
        return;
      }

      setIsImporting(true);
      setImportError(null);
      setItems((currentItems) =>
        currentItems.map((item) =>
          item.id === currentItem.id ? { ...item, status: 'importing' as const } : item,
        ),
      );

      try {
        const importResult = await ImportMod({
          GameID: gameID,
          Name: name,
          NewTags: tags.flatMap((tag) =>
            tag.ID === null ? [{ Color: tag.Color, Name: tag.Name }] : [],
          ),
          SourcePath: currentItem.sourcePath,
          SourceType: currentItem.sourceType,
          StrategyType: strategyType,
          TagIDs: tags.flatMap((tag) => (tag.ID === null ? [] : [tag.ID])),
          TargetRelativePath: targetRelativePath,
        });

        setLastImportSettings({
          strategyType,
          targetRelativePath,
        });

        let nextItems: ImportQueueItem[] = [];
        setItems((currentItems) => {
          nextItems = currentItems.map((item) =>
            item.id === currentItem.id
              ? {
                  ...item,
                  importedModName: importResult.Mod.Name,
                  status: 'imported' as const,
                }
              : item,
          );
          return nextItems;
        });

        try {
          await refreshMods();
        } catch (refreshError) {
          addErrorToast(refreshError);
        }

        addToast({
          message: `Imported ${importResult.Mod.Name}.`,
          tone: 'success',
        });

        advanceAfterSuccess(nextItems, importAnotherAfterComplete);
      } catch (error) {
        const message = getErrorMessage(error);
        setImportError(message);
        addErrorToast(error);
        setItems((currentItems) =>
          currentItems.map((item) =>
            item.id === currentItem.id
              ? { ...item, error: message, status: 'failed' as const }
              : item,
          ),
        );
      } finally {
        setIsImporting(false);
      }
    },
    [
      addErrorToast,
      addToast,
      advanceAfterSuccess,
      currentItem,
      gameID,
      importAnotherAfterComplete,
      isImporting,
      refreshMods,
    ],
  );

  const closeSummary = useCallback(() => {
    if (isBusy) {
      return;
    }

    setViewMode('idle');
    setItems([]);
    setCurrentItemID(null);
    setShowQueueChrome(false);
    setImportError(null);
    setImportAnotherAfterComplete(false);
    setPendingImportAnotherSourceType(null);
  }, [isBusy]);

  const closeQueue = useCallback(() => {
    if (isBusy) {
      return;
    }

    setViewMode('idle');
    setItems([]);
    setCurrentItemID(null);
    setShowQueueChrome(false);
    setImportError(null);
    setImportAnotherAfterComplete(false);
    setPendingImportAnotherSourceType(null);
  }, [isBusy]);

  const openQueue = useCallback(() => {
    if (items.length === 0) {
      return;
    }

    setViewMode('queue');
    setCurrentItemID(null);
  }, [items.length]);

  return {
    closeImportReview,
    closeQueue,
    closeSummary,
    currentItem,
    handleDroppedFiles,
    importAnotherAfterComplete,
    importCurrentItem,
    importError,
    isBusy,
    isEnqueueing,
    isImporting,
    isImportMenuOpen,
    isWizardOpen: viewMode === 'wizard' && currentItem !== null,
    items,
    lastImportSettings,
    openQueue,
    queuePosition,
    removeItem,
    reusePreviousImportSettings,
    reviewItem,
    setIsImportMenuOpen,
    setImportAnotherAfterComplete,
    setReusePreviousImportSettings,
    showQueueChrome,
    skipItem,
    startArchiveImportFlow,
    startFolderImportFlow,
    summaryCounts,
    viewMode,
  };
};

export type UseGameModImportQueueResult = ReturnType<typeof useGameModImportQueue>;
