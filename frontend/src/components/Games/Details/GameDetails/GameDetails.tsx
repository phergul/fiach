import { useCallback, useRef, useState } from 'react';

import { Link, useNavigate, useParams } from 'react-router-dom';
import { Archive, ArrowLeft, FolderOpen, Menu, Plus } from 'lucide-react';

import { ModSourceType } from '@bindings/github.com/phergul/fiach/internal/services/dto/models';
import { OpenDirectory } from '@bindings/github.com/phergul/fiach/internal/services/shellservice';
import { useToast } from '@components/Common/Toast/Toast';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { GameDetailsActionsMenu } from '@components/Games/Details/GameDetailsActionsMenu/GameDetailsActionsMenu';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import {
  GameDetailsTabs,
  type GameDetailsTab,
} from '@components/Games/Details/GameDetailsTabs/GameDetailsTabs';
import { GameDetailsMetadata } from '@components/Games/Details/Metadata/GameDetailsMetadata/GameDetailsMetadata';
import { GameModImportQueue } from '@components/Games/Details/Mods/GameModImportQueue/GameModImportQueue';
import { GameModImportQueueSummary } from '@components/Games/Details/Mods/GameModImportQueueSummary/GameModImportQueueSummary';
import { GameModImportWizard } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizard';
import { GameModUpdateModal } from '@components/Games/Details/Mods/GameModUpdateModal/GameModUpdateModal';
import { GameModsSection } from '@components/Games/Details/Mods/GameModsSection/GameModsSection';
import { GameProfilesSection } from '@components/Games/Details/Profiles/GameProfilesSection/GameProfilesSection';
import {
  useAppliedProfile,
  useGameArtwork,
  useGameModImportQueue,
  useGameModUpdateFlow,
  useGameMods,
  useModImportFileDrop,
  useGameOptiScaler,
  useGameProfiles,
  useGameReShade,
  useGameReShadeDetection,
  useGameStorageOverride,
  useStoredGames,
  useClickOutside,
} from '@hooks';

import './GameDetails.scss';

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

export const GameDetails = () => {
  const { gameId } = useParams();
  const navigate = useNavigate();
  const [activeTab, setActiveTab] = useState<GameDetailsTab>('profiles');
  const [isActionsMenuOpen, setIsActionsMenuOpen] = useState(false);
  const [isRestoreConfirmOpen, setIsRestoreConfirmOpen] = useState(false);
  const importMenuAnchorRef = useRef<HTMLDivElement>(null);
  const actionsMenuAnchorRef = useRef<HTMLDivElement>(null);
  const { games, isLoading, isScanning, loadError, retryLoadGames, updateStoredGame } =
    useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game =
    parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const appliedProfileManager = useAppliedProfile(game?.ID ?? null);
  const gameModManager = useGameMods(game?.ID ?? null);
  const reShadeDetection = useGameReShadeDetection(game?.ID ?? null);
  const reShade = useGameReShade(game?.ID ?? null);
  const optiScaler = useGameOptiScaler(game?.ID ?? null);
  const importQueue = useGameModImportQueue({
    gameID: game?.ID ?? null,
    refreshMods: gameModManager.refreshMods,
  });
  const handleDroppedFiles = useCallback(
    (files: string[]) => {
      void importQueue.handleDroppedFiles(files);
    },
    [importQueue.handleDroppedFiles],
  );
  useModImportFileDrop({
    enabled: activeTab === 'mods' && game !== undefined && !importQueue.isBusy,
    onFilesDropped: handleDroppedFiles,
  });
  const isImportMenuVisible =
    importQueue.isImportMenuOpen && game !== undefined && !importQueue.isBusy;
  const reuseFromPrevious =
    importQueue.queuePosition !== null &&
    importQueue.queuePosition.current > 1 &&
    importQueue.lastImportSettings !== null
      ? importQueue.lastImportSettings
      : null;
  useClickOutside(
    importMenuAnchorRef,
    () => importQueue.setIsImportMenuOpen(false),
    isImportMenuVisible,
  );
  useClickOutside(actionsMenuAnchorRef, () => setIsActionsMenuOpen(false), isActionsMenuOpen);
  const refreshAfterModUpdated = async () => {
    await Promise.all([
      gameModManager.refreshMods(),
      profileManager.refreshProfiles(),
      appliedProfileManager.refreshAppliedProfile(),
    ]);
  };
  const updateFlow = useGameModUpdateFlow({
    refreshAfterUpdate: refreshAfterModUpdated,
  });
  const { addErrorToast } = useToast();
  const storageOverride = useGameStorageOverride({
    game,
    onMenuClose: () => setIsActionsMenuOpen(false),
    updateStoredGame,
  });
  const { artworkSource: heroArtworkSource, handleArtworkError: handleHeroArtworkError } =
    useGameArtwork(game?.Source === 'steam' && game.SourceID ? game.SourceID : '', 'hero');
  const { artworkSource: logoArtworkSource, handleArtworkError: handleLogoArtworkError } =
    useGameArtwork(game?.Source === 'steam' && game.SourceID ? game.SourceID : '', 'logo');
  const isWaitingForGame = (isLoading || isScanning) && game === undefined;
  const hasLoadError = loadError !== null && game === undefined;
  const hasNotFound = !isWaitingForGame && !hasLoadError && game === undefined;
  const applyProfilePath = game === undefined ? '/library' : `/library/${game.ID}/apply`;
  const isRestorePending = appliedProfileManager.pendingAction === 'restore';
  const canRestoreVanilla =
    game !== undefined && appliedProfileManager.appliedProfile !== null && !isRestorePending;

  const openRestoreConfirm = () => {
    if (canRestoreVanilla) {
      setIsRestoreConfirmOpen(true);
    }
  };

  const closeRestoreConfirm = () => {
    if (!isRestorePending) {
      setIsRestoreConfirmOpen(false);
    }
  };

  const confirmRestoreVanilla = async () => {
    if (!canRestoreVanilla) {
      return;
    }

    try {
      const result = await appliedProfileManager.restoreVanilla();
      if (result.Success) {
        setIsRestoreConfirmOpen(false);
      }
    } catch {
      setIsRestoreConfirmOpen(false);
    }
  };

  const openInstallDirectory = async () => {
    if (game === undefined) {
      return;
    }

    setIsActionsMenuOpen(false);

    try {
      await OpenDirectory(game.InstallPath);
    } catch (error) {
      addErrorToast(error);
    }
  };

  const refreshAfterModDeleted = async () => {
    await Promise.all([
      profileManager.refreshProfiles(),
      appliedProfileManager.refreshAppliedProfile(),
    ]);
  };

  return (
    <section
      className={
        heroArtworkSource === '' ? 'game-details' : 'game-details game-details-with-backdrop'
      }
      aria-label="Game details"
    >
      <div className="game-details-toolbar">
        <Link className="game-details-back-link" to="/library">
          <ArrowLeft className="game-details-toolbar-icon" aria-hidden="true" />
          Back
        </Link>
        <div className="game-details-toolbar-actions">
          <div className="game-details-menu-anchor" ref={importMenuAnchorRef}>
            <button
              className="game-details-toolbar-button game-details-import-mods"
              disabled={game === undefined || importQueue.isBusy}
              onClick={() => {
                setIsActionsMenuOpen(false);
                importQueue.setIsImportMenuOpen((currentValue) => !currentValue);
              }}
              type="button"
              aria-expanded={importQueue.isImportMenuOpen}
            >
              <Plus className="game-details-toolbar-icon" aria-hidden="true" />
              <span>Import Mod</span>
            </button>

            <DropdownMenu
              ariaLabel="Import mod"
              isOpen={importQueue.isImportMenuOpen && game !== undefined && !importQueue.isBusy}
              items={[
                {
                  icon: FolderOpen,
                  label: 'Folder',
                  onSelect: importQueue.startFolderImportFlow,
                },
                {
                  icon: Archive,
                  label: 'ZIP Archive',
                  onSelect: importQueue.startArchiveImportFlow,
                },
              ]}
            />
          </div>
          <div className="game-details-actions-menu-anchor" ref={actionsMenuAnchorRef}>
            <button
              className="game-details-toolbar-button game-details-toolbar-icon-button"
              disabled={game === undefined}
              onClick={() => {
                importQueue.setIsImportMenuOpen(false);
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
                onClearStorageOverride={storageOverride.requestClearStorageOverride}
                onOpenInstallDirectory={openInstallDirectory}
                onOpenOptiScaler={() => {
                  setIsActionsMenuOpen(false);
                  navigate(`/library/${game.ID}/optiscaler`);
                }}
                onOpenReShade={() => {
                  setIsActionsMenuOpen(false);
                  navigate(`/library/${game.ID}/reshade`);
                }}
                onSetStorageOverride={storageOverride.requestSetStorageOverride}
              />
            )}
          </div>
        </div>
      </div>

      {heroArtworkSource !== '' && (
        <div className="game-details-backdrop" aria-hidden="true">
          <img
            className="game-details-backdrop-image"
            src={heroArtworkSource}
            alt=""
            onError={handleHeroArtworkError}
          />
        </div>
      )}

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
            logoArtworkSource={logoArtworkSource}
            onLogoArtworkError={handleLogoArtworkError}
          />

          <GameDetailsMetadata
            isStorageUsageLoading={gameModManager.isStorageUsageLoading}
            modCount={gameModManager.mods.length}
            optiScaler={optiScaler}
            profileCount={profileManager.profiles.length}
            reShade={reShade}
            reShadeDetection={reShadeDetection}
            storageUsedBytes={gameModManager.storageUsedBytes}
          />

          <GameDetailsTabs activeTab={activeTab} onActiveTabChange={setActiveTab} />

          {activeTab === 'mods' ? (
            <GameModsSection
              isFileDropTargetActive={activeTab === 'mods' && !importQueue.isBusy}
              isImportDisabled={importQueue.isBusy}
              isUpdateDisabled={updateFlow.isBusy}
              modManager={gameModManager}
              onModDeleted={refreshAfterModDeleted}
              onImportArchive={importQueue.startArchiveImportFlow}
              onImportFolder={importQueue.startFolderImportFlow}
              onUpdateArchiveMod={updateFlow.startArchiveUpdateFlow}
              onUpdateFolderMod={updateFlow.startFolderUpdateFlow}
            />
          ) : (
            <GameProfilesSection
              appliedProfileManager={appliedProfileManager}
              applyProfilePath={applyProfilePath}
              gameID={game.ID}
              gameModManager={gameModManager}
              onRestoreVanilla={openRestoreConfirm}
              profileManager={profileManager}
            />
          )}
        </>
      )}

      <GameModImportWizard
        availableTags={gameModManager.gameTags}
        error={importQueue.importError}
        gameID={game?.ID ?? 0}
        initialName={importQueue.currentItem?.initialName ?? ''}
        isBusy={importQueue.isImporting}
        isOpen={importQueue.isWizardOpen}
        onClose={importQueue.closeImportReview}
        onImport={importQueue.importCurrentItem}
        onImportAnotherAfterCompleteChange={importQueue.setImportAnotherAfterComplete}
        importAnotherAfterComplete={importQueue.importAnotherAfterComplete}
        onReusePreviousSettingsChange={importQueue.setReusePreviousImportSettings}
        queuePosition={importQueue.queuePosition}
        reuseFromPrevious={reuseFromPrevious}
        reusePreviousSettings={importQueue.reusePreviousImportSettings}
        sourceLabel={importQueue.currentItem?.sourceLabel ?? 'Source'}
        sourcePath={importQueue.currentItem?.sourcePath ?? ''}
        sourceType={importQueue.currentItem?.sourceType ?? ModSourceType.$zero}
        suggestedStrategyType={importQueue.currentItem?.suggestedStrategy ?? null}
        targetPath={importQueue.currentItem?.targetPath ?? ''}
      />

      <GameModImportQueue
        isBusy={importQueue.isBusy}
        isOpen={importQueue.viewMode === 'queue'}
        items={importQueue.items}
        onClose={importQueue.closeQueue}
        onRemoveItem={importQueue.removeItem}
        onReviewItem={importQueue.reviewItem}
        onSkipItem={importQueue.skipItem}
      />

      <GameModImportQueueSummary
        counts={importQueue.summaryCounts}
        isBusy={importQueue.isBusy}
        isOpen={importQueue.viewMode === 'summary'}
        items={importQueue.items}
        onClose={importQueue.closeSummary}
      />

      <GameModUpdateModal
        error={updateFlow.updateError}
        isBusy={updateFlow.isUpdatingMod}
        isOpen={updateFlow.updateReview !== null}
        onClose={updateFlow.closeUpdateReview}
        onConfirm={updateFlow.confirmUpdateMod}
        result={updateFlow.updateReview?.preview ?? null}
      />

      <ConfirmDialog
        confirmLabel={storageOverride.pendingStorageOverride?.confirmLabel}
        confirmTone="default"
        isBusy={storageOverride.isApplyingStorageOverride}
        isOpen={storageOverride.pendingStorageOverride !== null}
        message="Changing this setting affects future imports only. Existing imported mod folders and mod rows will not be moved."
        onCancel={storageOverride.cancelStorageOverride}
        onConfirm={storageOverride.applyStorageOverride}
        title={storageOverride.pendingStorageOverride?.title ?? 'Confirm Storage Change'}
      />

      <ConfirmDialog
        cancelLabel="Cancel"
        confirmLabel="Restore Vanilla"
        confirmTone="warning"
        isBusy={isRestorePending}
        isOpen={isRestoreConfirmOpen}
        message={
          appliedProfileManager.appliedProfile === null
            ? 'No profile is currently applied.'
            : `Restore vanilla files for ${game?.Name ?? 'this game'}? This will change the installed game files directly and revert ${appliedProfileManager.appliedProfile.ProfileName}.`
        }
        onCancel={closeRestoreConfirm}
        onConfirm={confirmRestoreVanilla}
        title="Restore Vanilla"
      />
    </section>
  );
};
