import './GameModsSection.scss';

export const GameModsSection = () => {
  return (
    <section className="game-mods-section" aria-labelledby="game-mods-section-title">
      <div className="game-mods-section-header">
        <h2 className="game-mods-section-title" id="game-mods-section-title">
          Imported mods
        </h2>
        <p className="game-mods-section-count">0 mods</p>
      </div>

      <p className="game-mods-section-empty">Imported mods for this game will appear here.</p>
    </section>
  );
};
