import './GameDetailsTabs.scss';

export type GameDetailsTab = 'mods' | 'profiles';

interface GameDetailsTabsProps {
  activeTab: GameDetailsTab;
  onActiveTabChange: (tab: GameDetailsTab) => void;
}

export const GameDetailsTabs = ({
  activeTab,
  onActiveTabChange,
}: GameDetailsTabsProps) => {
  return (
    <div className="game-details-tabs" role="tablist" aria-label="Game detail sections">
      <button
        className={
          activeTab === 'mods'
            ? 'game-details-tabs-tab game-details-tabs-tab-active'
            : 'game-details-tabs-tab'
        }
        onClick={() => onActiveTabChange('mods')}
        role="tab"
        type="button"
        aria-selected={activeTab === 'mods'}
      >
        Mods
      </button>
      <button
        className={
          activeTab === 'profiles'
            ? 'game-details-tabs-tab game-details-tabs-tab-active'
            : 'game-details-tabs-tab'
        }
        onClick={() => onActiveTabChange('profiles')}
        role="tab"
        type="button"
        aria-selected={activeTab === 'profiles'}
      >
        Profiles
      </button>
    </div>
  );
};
