import type { ReactNode } from 'react';

import './StateBlock.scss';

interface StateBlockProps {
  children?: ReactNode;
  className?: string;
  message?: ReactNode;
  title?: ReactNode;
}

export const StateBlock = ({ children, className, message, title }: StateBlockProps) => {
  const stateBlockClassName = className === undefined ? 'state-block' : `state-block ${className}`;

  return (
    <div className={stateBlockClassName}>
      {title !== undefined && <p className="state-block-title">{title}</p>}
      {message !== undefined && <p className="state-block-message">{message}</p>}
      {children}
    </div>
  );
};
