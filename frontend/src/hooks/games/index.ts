// artwork
export { useGameArtwork } from './artwork/useGameArtwork';

// mods
export { useGameModImportFlow } from './mods/useGameModImportFlow';
export type { UseGameModImportFlowResult } from './mods/useGameModImportFlow';
export { useGameModUpdateFlow } from './mods/useGameModUpdateFlow';
export type { UseGameModUpdateFlowResult } from './mods/useGameModUpdateFlow';
export { useGameMods } from './mods/useGameMods';
export type { UseGameModsResult } from './mods/useGameMods';

// optiscaler
export { getOptiScalerAggregateStatus, useGameOptiScaler } from './optiscaler/useGameOptiScaler';
export type {
  OptiScalerAggregateStatus,
  UseGameOptiScalerResult,
} from './optiscaler/useGameOptiScaler';

// profiles
export {
  fetchGameProfiles,
  invalidateGameProfiles,
  preloadGameProfiles,
  useGameProfiles,
} from './profiles/useGameProfiles';
export type { CachedGameProfiles, UseGameProfilesResult } from './profiles/useGameProfiles';

// reshade
export {
  getReShadeAggregateStatus,
  isReShadeUpdateAvailable,
  useGameReShade,
} from './reshade/useGameReShade';
export type { ReShadeAggregateStatus, UseGameReShadeResult } from './reshade/useGameReShade';
export { useGameReShadeDetection } from './reshade/useGameReShadeDetection';
export type { UseGameReShadeDetectionResult } from './reshade/useGameReShadeDetection';

// search
export { useGameSearch } from './search/useGameSearch';

// storage
export { useGameStorageOverride } from './storage/useGameStorageOverride';
export type { UseGameStorageOverrideResult } from './storage/useGameStorageOverride';

// stored
export {
  fetchStoredGames,
  invalidateStoredGames,
  preloadStoredGames,
  useStoredGames,
} from './stored/useStoredGames';
