import './GameModListHeader.scss';

export const GameModListHeader = () => {
  return (
    <div className="game-mod-list-header" aria-hidden="true">
      <span className="game-mod-list-header-label">Mod</span>
      <span className="game-mod-list-header-label">Tags</span>
      <div className="game-mod-list-header-metadata">
        <span className="game-mod-list-header-label">Source</span>
        <span className="game-mod-list-header-label">Contents</span>
        <span className="game-mod-list-header-label">Size</span>
      </div>
    </div>
  );
};
