import { useState } from 'react';

import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Plus, Menu } from 'lucide-react';

import { ImportModFolder } from '@bindings/github.com/phergul/mod-manager/internal/services/modservice';
import {
  ResolveGameModStoragePath,
  SetGameModStoragePathOverride,
} from '@bindings/github.com/phergul/mod-manager/internal/services/settingsservice';
import type { StoredGame } from '@bindings/github.com/phergul/mod-manager/internal/storage/models';
import { ImageType } from '@bindings/github.com/phergul/mod-manager/internal/steam/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
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
import { getErrorMessage, openDirectory } from '@utils';

import './GameDetails.scss';

interface ImportReviewState {
  initialName: string;
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

export const GameDetails = () => {
  const { gameId } = useParams();
  const { addToast } = useToast();
  const [activeTab, setActiveTab] = useState<GameDetailsTab>('profiles');
  const [importReview, setImportReview] = useState<ImportReviewState | null>(null);
  const [importError, setImportError] = useState<string | null>(null);
  const [isImporting, setIsImporting] = useState(false);
  const [isActionsMenuOpen, setIsActionsMenuOpen] = useState(false);
  const [pendingStorageOverride, setPendingStorageOverride] = useState<PendingStorageOverride | null>(null);
  const [isApplyingStorageOverride, setIsApplyingStorageOverride] = useState(false);
  const { games, isLoading, isScanning, loadError, retryLoadGames, updateStoredGame } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const gameModManager = useGameMods(game?.ID ?? null);
  const heroArtworkSource = useGameArtwork(
    game?.Source === 'steam' && game.SourceID ? game.SourceID : '',
    ImageType.ImageTypeHero,
  );
  const logoArtworkSource = useGameArtwork(
    game?.Source === 'steam' && game.SourceID ? game.SourceID : '',
    ImageType.ImageTypeLogo,
  );
  const isWaitingForGame = (isLoading || isScanning) && game === undefined;
  const hasLoadError = loadError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;

  const startImportFlow = async () => {
    if (game === undefined || isImporting) {
      return;
    }

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
      const importedMod = await ImportModFolder(game.ID, name, importReview.sourcePath);
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
          <button
            className="game-details-toolbar-button game-details-import-mods"
            disabled={game === undefined || isImporting}
            onClick={startImportFlow}
            type="button"
          >
            <Plus className="game-details-toolbar-icon" aria-hidden="true" />
            <span>Import Mods</span>
          </button>
          <div className="game-details-actions-menu-anchor">
            <button
              className="game-details-toolbar-button game-details-toolbar-icon-button"
              disabled={game === undefined}
              onClick={() => setIsActionsMenuOpen((currentValue) => !currentValue)}
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
            logoArtworkSource={logoArtworkSource}
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
              onImportMod={startImportFlow}
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
