import { Link } from 'react-router-dom';

import { StateBlock } from '@components/Common/StateBlock/StateBlock';

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
    return <StateBlock className="game-details-state" message="Loading game..." />;
  }

  return (
    <StateBlock className="game-details-state" title={title} message={message}>
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
    </StateBlock>
  );
};
