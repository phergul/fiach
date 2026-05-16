import { useState } from 'react';

import { Link, useParams } from 'react-router-dom';
import { Archive, ArrowLeft, FolderOpen, Menu, Plus } from 'lucide-react';

import { ImportModArchive, ImportModFolder } from '@bindings/github.com/phergul/mod-manager/internal/services/modservice';
import {
  ResolveGameModStoragePath,
  SetGameModStoragePathOverride,
} from '@bindings/github.com/phergul/mod-manager/internal/services/settingsservice';
import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { useToast } from '@components/Common/Toast/Toast';
import { GameDetailsActionsMenu } from '@components/Games/Details/GameDetailsActionsMenu/GameDetailsActionsMenu';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { GameDetailsTabs, type GameDetailsTab } from '@components/Games/Details/GameDetailsTabs/GameDetailsTabs';
import { GameDetailsMetadata } from '@components/Games/Details/Metadata/GameDetailsMetadata/GameDetailsMetadata';
import { GameModImportReviewDialog } from '@components/Games/Details/Mods/GameModImportReviewDialog/GameModImportReviewDialog';
import { GameModsSection } from '@components/Games/Details/Mods/GameModsSection/GameModsSection';
import { GameProfilesSection } from '@components/Games/Details/Profiles/GameProfilesSection/GameProfilesSection';
import { useGameArtwork, useGameMods, useGameProfiles, useStoredGames } from '@hooks';
import { getErrorMessage, openDirectory, openZipArchive } from '@utils';

import './GameDetails.scss';

interface ImportReviewState {
  initialName: string;
  sourceKind: 'folder' | 'archive';
  sourceLabel: string;
  sourcePath: string;
  targetPath: string;
}

interface PendingStorageOverride {
  confirmLabel: string;
  path: string;
  successMessage: string;
  title: string;
}

const parseGameID = (gameID: string | undefined) => {
  if (gameID === undefined || gameID.trim() === '') {
    return null;
  }

  const parsedGameID = Number(gameID);
  if (!Number.isInteger(parsedGameID) || parsedGameID <= 0) {
    return null;
  }

  return parsedGameID;
};

const getFolderName = (path: string) => {
  const trimmedPath = path.trim().replace(/[\\/]+$/, '');
  const folderName = trimmedPath.split(/[\\/]/).pop();

  return folderName && folderName.trim() !== '' ? folderName : 'Imported Mod';
};

const getArchiveName = (path: string) => {
  const fileName = getFolderName(path);
  const archiveName = fileName.replace(/\.zip$/i, '');

  return archiveName.trim() === '' ? 'Imported Mod' : archiveName;
};

export const GameDetails = () => {
  const { gameId } = useParams();
  const { addToast } = useToast();
  const [activeTab, setActiveTab] = useState<GameDetailsTab>('profiles');
  const [importReview, setImportReview] = useState<ImportReviewState | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const [isImportMenuOpen, setIsImportMenuOpen] = useState(false);
  const [isActionsMenuOpen, setIsActionsMenuOpen] = useState(false);
  const [pendingStorageOverride, setPendingStorageOverride] = useState<PendingStorageOverride | null>(null);
  const [isApplyingStorageOverride, setIsApplyingStorageOverride] = useState(false);
  const { games, isLoading, isScanning, loadError, retryLoadGames, updateStoredGame } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const gameModManager = useGameMods(game?.ID ?? null);
  const {
    artworkSource: heroArtworkSource,
    handleArtworkError: handleHeroArtworkError,
  } = useGameArtwork(
    game?.Source === 'steam' && game.SourceID ? game.SourceID : '',
    'hero',
  );
  const {
    artworkSource: logoArtworkSource,
    handleArtworkError: handleLogoArtworkError,
  } = useGameArtwork(
    game?.Source === 'steam' && game.SourceID ? game.SourceID : '',
    'logo',
  );
  const isWaitingForGame = (isLoading || isScanning) && game === undefined;
  const hasLoadError = loadError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;

  const startFolderImportFlow = async () => {
    if (game === undefined || isImporting) {
      return;
    }

    setIsImportMenuOpen(false);
    setImportError(null);

    try {
      const sourcePath = await openDirectory({
        buttonText: 'Review Import',
        title: 'Select mod folder',
      });
      if (sourcePath === null) {
        return;
      }

      const targetPath = await ResolveGameModStoragePath(game.ID);
      setImportReview({
        initialName: getFolderName(sourcePath),
        sourceKind: 'folder',
        sourceLabel: 'Source folder',
        sourcePath,
        targetPath,
      });
    } catch (error) {
      const message = getErrorMessage(error);
      addToast({
        message,
        tone: 'error',
      });
    }
  };

  const startArchiveImportFlow = async () => {
    if (game === undefined || isImporting) {
      return;
    }

    setIsImportMenuOpen(false);
    setImportError(null);

    try {
      const sourcePath = await openZipArchive({
        buttonText: 'Review Import',
        title: 'Select mod archive',
      });
      if (sourcePath === null) {
        return;
      }

      const targetPath = await ResolveGameModStoragePath(game.ID);
      setImportReview({
        initialName: getArchiveName(sourcePath),
        sourceKind: 'archive',
        sourceLabel: 'Source archive',
        sourcePath,
        targetPath,
      });
    } catch (error) {
      const message = getErrorMessage(error);
      addToast({
        message,
        tone: 'error',
      });
    }
  };

  const closeImportReview = () => {
    if (isImporting) {
      return;
    }

    setImportReview(null);
    setImportError(null);
  };

  const importReviewedMod = async (name: string) => {
    if (game === undefined || importReview === null || isImporting) {
      return;
    }

    setIsImporting(true);
    setImportError(null);

    try {
      const importedMod = importReview.sourceKind === 'archive'
        ? await ImportModArchive(game.ID, name, importReview.sourcePath)
        : await ImportModFolder(game.ID, name, importReview.sourcePath);
      setImportReview(null);

      try {
        await gameModManager.refreshMods();
      } catch (refreshError) {
        addToast({
          message: getErrorMessage(refreshError),
          tone: 'error',
        });
      }

      addToast({
        message: `Imported ${importedMod.Name}.`,
        tone: 'success',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      setImportError(message);
      addToast({
        message,
        tone: 'error',
      });
    } finally {
      setIsImporting(false);
    }
  };

  const requestSetStorageOverride = async () => {
    if (game === undefined || isApplyingStorageOverride) {
      return;
    }

    setIsActionsMenuOpen(false);

    try {
      const path = await openDirectory({
        buttonText: 'Use Folder',
        canCreateDirectories: true,
        title: 'Select mod storage override',
      });
      if (path === null) {
        return;
      }

      setPendingStorageOverride({
        confirmLabel: 'Set Override',
        path,
        successMessage: 'Mod storage override set.',
        title: 'Set Storage Override?',
      });
    } catch (error) {
      const message = getErrorMessage(error);
      addToast({
        message,
        tone: 'error',
      });
    }
  };

  const requestClearStorageOverride = () => {
    if (game === undefined || isApplyingStorageOverride) {
      return;
    }

    setIsActionsMenuOpen(false);
    setPendingStorageOverride({
      confirmLabel: 'Clear Override',
      path: '',
      successMessage: 'Mod storage override cleared.',
      title: 'Clear Storage Override?',
    });
  };

  const applyStorageOverride = async () => {
    if (game === undefined || pendingStorageOverride === null || isApplyingStorageOverride) {
      return;
    }

    setIsApplyingStorageOverride(true);

    try {
      const updatedGame: StoredGame = await SetGameModStoragePathOverride(game.ID, pendingStorageOverride.path);
      updateStoredGame(updatedGame);
      addToast({
        message: pendingStorageOverride.successMessage,
        tone: 'success',
      });
      setPendingStorageOverride(null);
    } catch (error) {
      const message = getErrorMessage(error);
      addToast({
        message,
        tone: 'error',
      });
    } finally {
      setIsApplyingStorageOverride(false);
    }
  };

  return (
    <section
      className={heroArtworkSource === '' ? 'game-details' : 'game-details game-details-with-backdrop'}
      aria-label="Game details"
    >
      <div className="game-details-toolbar">
        <Link className="game-details-back-link" to="/library">
          <ArrowLeft className="game-details-toolbar-icon" aria-hidden="true" />
          Back
        </Link>
        <div className="game-details-toolbar-actions">
          <div className="game-details-menu-anchor">
            <button
              className="game-details-toolbar-button game-details-import-mods"
              disabled={game === undefined || isImporting}
              onClick={() => {
                setIsActionsMenuOpen(false);
                setIsImportMenuOpen((currentValue) => !currentValue);
              }}
              type="button"
              aria-expanded={isImportMenuOpen}
            >
              <Plus className="game-details-toolbar-icon" aria-hidden="true" />
              <span>Import Mod</span>
            </button>

            <DropdownMenu
              ariaLabel="Import mod"
              isOpen={isImportMenuOpen && game !== undefined && !isImporting}
              items={[
                {
                  icon: FolderOpen,
                  label: 'Folder',
                  onSelect: startFolderImportFlow,
                },
                {
                  icon: Archive,
                  label: 'ZIP Archive',
                  onSelect: startArchiveImportFlow,
                },
              ]}
            />
          </div>
          <div className="game-details-actions-menu-anchor">
            <button
              className="game-details-toolbar-button game-details-toolbar-icon-button"
              disabled={game === undefined}
              onClick={() => {
                setIsImportMenuOpen(false);
                setIsActionsMenuOpen((currentValue) => !currentValue);
              }}
              title="Game actions"
              type="button"
              aria-expanded={isActionsMenuOpen}
            >
              <Menu className="game-details-toolbar-icon" aria-hidden="true" />
            </button>

            {game !== undefined && (
              <GameDetailsActionsMenu
                game={game}
                isOpen={isActionsMenuOpen}
                onClearStorageOverride={requestClearStorageOverride}
                onSetStorageOverride={requestSetStorageOverride}
              />
            )}
          </div>
        </div>
      </div>

      {isWaitingForGame && <GameDetailsState />}

      {hasLoadError && (
        <GameDetailsState
          actionLabel="Retry"
          linkLabel="Return to library"
          message={loadError}
          onAction={retryLoadGames}
          title="Could not load game."
        />
      )}

      {hasNotFound && (
        <GameDetailsState
          message="This game is not currently available in the library."
          title="Game not found."
        />
      )}

      {game !== undefined && (
        <>
          <GameDetailsHeader
            game={game}
            heroArtworkSource={heroArtworkSource}
            onHeroArtworkError={handleHeroArtworkError}
            logoArtworkSource={logoArtworkSource}
            onLogoArtworkError={handleLogoArtworkError}
          />

          <GameDetailsMetadata
            game={game}
            modCount={gameModManager.mods.length}
            profileCount={profileManager.profiles.length}
            profileModsByProfileID={profileManager.profileModsByProfileID}
          />

          <GameDetailsTabs activeTab={activeTab} onActiveTabChange={setActiveTab} />

          {activeTab === 'mods' ? (
            <GameModsSection
              isImportDisabled={isImporting}
              modManager={gameModManager}
              onImportArchive={startArchiveImportFlow}
              onImportFolder={startFolderImportFlow}
            />
          ) : (
            <GameProfilesSection gameModManager={gameModManager} profileManager={profileManager} />
          )}
        </>
      )}

      <GameModImportReviewDialog
        error={importError}
        initialName={importReview?.initialName ?? ''}
        isBusy={isImporting}
        isOpen={importReview !== null}
        onClose={closeImportReview}
        onImport={importReviewedMod}
        sourceLabel={importReview?.sourceLabel ?? 'Source'}
        sourcePath={importReview?.sourcePath ?? ''}
        targetPath={importReview?.targetPath ?? ''}
      />

      <ConfirmDialog
        confirmLabel={pendingStorageOverride?.confirmLabel}
        confirmTone="default"
        isBusy={isApplyingStorageOverride}
        isOpen={pendingStorageOverride !== null}
        message="Changing this setting affects future imports only. Existing imported mod folders and mod rows will not be moved."
        onCancel={() => {
          if (!isApplyingStorageOverride) {
            setPendingStorageOverride(null);
          }
        }}
        onConfirm={applyStorageOverride}
        title={pendingStorageOverride?.title ?? 'Confirm Storage Change'}
      />
    </section>
  );
};
