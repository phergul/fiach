// common
export { useClickOutside, useWindowMaximised } from './common';

// deployment
export {
  fetchDeploymentReviewPreview,
  invalidateDeploymentPreview,
  useDeploymentFileDetail,
  useDeploymentFileInspection,
  useDeploymentReviewPreview,
  useDeploymentTree,
  preloadDeploymentReviewPreview,
  type UseDeploymentFileDetailResult,
  type UseDeploymentFileInspectionResult,
  type UseDeploymentReviewPreviewResult,
  type UseDeploymentTreeResult,
} from './deployment';

// games
export {
  fetchGameProfiles,
  fetchStoredGames,
  getOptiScalerAggregateStatus,
  getReShadeAggregateStatus,
  invalidateGameProfiles,
  invalidateStoredGames,
  isReShadeUpdateAvailable,
  useGameArtwork,
  useGameModImportQueue,
  useGameMods,
  useGameModUpdateFlow,
  useModImportFileDrop,
  useGameOptiScaler,
  useGameProfiles,
  useGameReShade,
  useGameReShadeDetection,
  useGameSearch,
  useGameStorageOverride,
  useStoredGames,
  preloadGameProfiles,
  preloadStoredGames,
  type OptiScalerAggregateStatus,
  type ReShadeAggregateStatus,
  type ImportQueueItem,
  type UseGameModImportQueueResult,
  type UseGameModsResult,
  type UseGameModUpdateFlowResult,
  type UseGameOptiScalerResult,
  type UseGameProfilesResult,
  type UseGameReShadeDetectionResult,
  type UseGameReShadeResult,
  type UseGameStorageOverrideResult,
} from './games';

// profiles
export {
  fetchAppliedProfile,
  invalidateAppliedProfile,
  preloadAppliedProfile,
  useAppliedProfile,
  type UseAppliedProfileResult,
} from './profiles';

// runtime
export { useRuntime } from './runtime';
