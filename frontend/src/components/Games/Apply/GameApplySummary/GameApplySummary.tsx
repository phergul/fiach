import './GameApplySummary.scss';

export interface GameApplySummaryItem {
  label: string;
  value: number;
}

interface GameApplySummaryProps {
  items: GameApplySummaryItem[];
}

export const GameApplySummary = ({ items }: GameApplySummaryProps) => {
  return (
    <dl className="game-apply-summary" aria-label="Operation plan summary">
      {items.map((item) => (
        <div className="game-apply-summary-item" key={item.label}>
          <dt className="game-apply-summary-label">{item.label}</dt>
          <dd className="game-apply-summary-value">{item.value}</dd>
        </div>
      ))}
    </dl>
  );
};
