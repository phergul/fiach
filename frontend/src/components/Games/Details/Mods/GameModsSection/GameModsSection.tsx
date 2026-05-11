import './GameModsSection.scss';

export const GameModsSection = () => {
  return (
    <section className="game-mods-section" aria-label="Imported mods">
      <div className="game-mods-section-search">
        <input
          className="game-mods-section-search-input"
          disabled
          placeholder="Search mods..."
          type="search"
        />
      </div>

      <p className="game-mods-section-empty">Imported mods for this game will appear here.</p>
    </section>
  );
};
