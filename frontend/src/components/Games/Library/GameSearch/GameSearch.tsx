import './GameSearch.scss';

interface GameSearchProps {
  onSearchQueryChange: (searchQuery: string) => void;
  searchQuery: string;
}

export const GameSearch = ({ onSearchQueryChange, searchQuery }: GameSearchProps) => {
  return (
    <label className="game-search">
      <span className="game-search-label">Search</span>
      <input
        className="game-search-input"
        onChange={(event) => onSearchQueryChange(event.target.value)}
        placeholder="Search games"
        type="search"
        value={searchQuery}
      />
    </label>
  );
};
