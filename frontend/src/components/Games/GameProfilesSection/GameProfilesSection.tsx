import './GameProfilesSection.scss';

export const GameProfilesSection = () => {
  return (
    <section className="game-profiles-section" aria-labelledby="game-profiles-section-title">
      <div className="game-profiles-section-header">
        <h2 className="game-profiles-section-title" id="game-profiles-section-title">
          Profiles
        </h2>
        <p className="game-profiles-section-count">0 profiles</p>
      </div>

      <p className="game-profiles-section-empty">Profiles for this game will appear here.</p>
    </section>
  );
};
