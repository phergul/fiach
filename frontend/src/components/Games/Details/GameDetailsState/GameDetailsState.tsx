import { Link } from 'react-router-dom';

import './GameDetailsState.scss';

interface GameDetailsStateProps {
  actionLabel?: string;
  linkLabel?: string;
  message?: string;
  title?: string;
  onAction?: () => void;
}

export const GameDetailsState = ({
  actionLabel,
  linkLabel,
  message,
  title,
  onAction,
}: GameDetailsStateProps) => {
  if (title === undefined && message === undefined) {
    return <p className="game-details-state">Loading game...</p>;
  }

  return (
    <div className="game-details-state">
      {title !== undefined && <p className="game-details-state-title">{title}</p>}
      {message !== undefined && <p className="game-details-state-message">{message}</p>}
      {(actionLabel !== undefined || linkLabel !== undefined) && (
        <div className="game-details-state-actions">
          {actionLabel !== undefined && onAction !== undefined && (
            <button className="game-details-state-action" onClick={onAction} type="button">
              {actionLabel}
            </button>
          )}
          {linkLabel !== undefined && (
            <Link className="game-details-state-link" to="/library">
              {linkLabel}
            </Link>
          )}
        </div>
      )}
    </div>
  );
};
