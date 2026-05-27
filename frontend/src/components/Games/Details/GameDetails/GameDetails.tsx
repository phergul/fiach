import { useState } from 'react';

import { Link, useParams } from 'react-router-dom';
import { Archive, ArrowLeft, FolderOpen, Menu, Plus, RotateCcw } from 'lucide-react';

import { ModSourceType } from '@bindings/github.com/phergul/mod-manager/internal/services/dto/models';
import { ConfirmDialog } from '@components/Common/ConfirmDialog/ConfirmDialog';
import { DropdownMenu } from '@components/Common/DropdownMenu/DropdownMenu';
import { GameDetailsActionsMenu } from '@components/Games/Details/GameDetailsActionsMenu/GameDetailsActionsMenu';
import { GameDetailsHeader } from '@components/Games/Details/GameDetailsHeader/GameDetailsHeader';
import { GameDetailsState } from '@components/Games/Details/GameDetailsState/GameDetailsState';
import { GameDetailsTabs, type GameDetailsTab } from '@components/Games/Details/GameDetailsTabs/GameDetailsTabs';
import { GameDetailsMetadata } from '@components/Games/Details/Metadata/GameDetailsMetadata/GameDetailsMetadata';
import { GameModImportWizard } from '@components/Games/Details/Mods/GameModImportWizard/GameModImportWizard';
import { GameModsSection } from '@components/Games/Details/Mods/GameModsSection/GameModsSection';
import { GameProfilesSection } from '@components/Games/Details/Profiles/GameProfilesSection/GameProfilesSection';
import {
  useAppliedProfile,
  useGameArtwork,
  useGameModImportFlow,
  useGameMods,
  useGameProfiles,
  useGameStorageOverride,
  useStoredGames,
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
  const [activeTab, setActiveTab] = useState<GameDetailsTab>('profiles');
  const [isActionsMenuOpen, setIsActionsMenuOpen] = useState(false);
  const [isRestoreConfirmOpen, setIsRestoreConfirmOpen] = useState(false);
  const { games, isLoading, isScanning, loadError, retryLoadGames, updateStoredGame } = useStoredGames();
  const parsedGameID = parseGameID(gameId);
  const game = parsedGameID === null ? undefined : games.find((storedGame) => storedGame.ID === parsedGameID);
  const profileManager = useGameProfiles(game?.ID ?? null);
  const appliedProfileManager = useAppliedProfile(game?.ID ?? null);
  const gameModManager = useGameMods(game?.ID ?? null);
  const importFlow = useGameModImportFlow({
    gameID: game?.ID ?? null,
    refreshMods: gameModManager.refreshMods,
  });
  const storageOverride = useGameStorageOverride({
    game,
    onMenuClose: () => setIsActionsMenuOpen(false),
    updateStoredGame,
  });
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
  const applyProfilePath = game === undefined ? '/library' : `/library/${game.ID}/apply`;
  const isRestorePending = appliedProfileManager.pendingAction === 'restore';
  const canRestoreVanilla = game !== undefined && appliedProfileManager.appliedProfile !== null && !isRestorePending;

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
            className="game-details-toolbar-button game-details-restore-vanilla"
            disabled={!canRestoreVanilla}
            onClick={openRestoreConfirm}
            title={
              appliedProfileManager.appliedProfile === null
                ? 'No profile is applied.'
                : `Restore vanilla from ${appliedProfileManager.appliedProfile.ProfileName}.`
            }
            type="button"
          >
            <RotateCcw className="game-details-toolbar-icon" aria-hidden="true" />
            <span>Restore Vanilla</span>
          </button>

          <div className="game-details-menu-anchor">
            <button
              className="game-details-toolbar-button game-details-import-mods"
              disabled={game === undefined || importFlow.isImporting}
              onClick={() => {
                setIsActionsMenuOpen(false);
                importFlow.setIsImportMenuOpen((currentValue) => !currentValue);
              }}
              type="button"
              aria-expanded={importFlow.isImportMenuOpen}
            >
              <Plus className="game-details-toolbar-icon" aria-hidden="true" />
              <span>Import Mod</span>
            </button>

            <DropdownMenu
              ariaLabel="Import mod"
              isOpen={importFlow.isImportMenuOpen && game !== undefined && !importFlow.isImporting}
              items={[
                {
                  icon: FolderOpen,
                  label: 'Folder',
                  onSelect: importFlow.startFolderImportFlow,
                },
                {
                  icon: Archive,
                  label: 'ZIP Archive',
                  onSelect: importFlow.startArchiveImportFlow,
                },
              ]}
            />
          </div>
          <div className="game-details-actions-menu-anchor">
            <button
              className="game-details-toolbar-button game-details-toolbar-icon-button"
              disabled={game === undefined}
              onClick={() => {
                importFlow.setIsImportMenuOpen(false);
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
            game={game}
            isStorageUsageLoading={gameModManager.isStorageUsageLoading}
            modCount={gameModManager.mods.length}
            profileCount={profileManager.profiles.length}
            storageUsedBytes={gameModManager.storageUsedBytes}
          />

          <GameDetailsTabs activeTab={activeTab} onActiveTabChange={setActiveTab} />

          {activeTab === 'mods' ? (
            <GameModsSection
              isImportDisabled={importFlow.isImporting}
              modManager={gameModManager}
              onImportArchive={importFlow.startArchiveImportFlow}
              onImportFolder={importFlow.startFolderImportFlow}
            />
          ) : (
            <GameProfilesSection
              appliedProfileManager={appliedProfileManager}
              applyProfilePath={applyProfilePath}
              gameModManager={gameModManager}
              onRestoreVanilla={openRestoreConfirm}
              profileManager={profileManager}
            />
          )}
        </>
      )}

      <GameModImportWizard
        error={importFlow.importError}
        gameID={game?.ID ?? 0}
        initialName={importFlow.importWizard?.initialName ?? ''}
        isBusy={importFlow.isImporting}
        isOpen={importFlow.importWizard !== null}
        onClose={importFlow.closeImportReview}
        onImport={importFlow.importWizardMod}
        sourceLabel={importFlow.importWizard?.sourceLabel ?? 'Source'}
        sourcePath={importFlow.importWizard?.sourcePath ?? ''}
        sourceType={importFlow.importWizard?.sourceType ?? ModSourceType.$zero}
        targetPath={importFlow.importWizard?.targetPath ?? ''}
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
        confirmTone="default"
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
