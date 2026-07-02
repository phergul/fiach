import { Copy, Minus, Square, X } from 'lucide-react';
import { Window } from '@wailsio/runtime';

import appIcon from '@assets/app-icon.png';
import { useWindowMaximised } from '@hooks';

import './TitleBar.scss';

interface TitleBarProps {
  title: string;
}

export const TitleBar = ({ title }: TitleBarProps) => {
  const { isMaximised, toggleMaximised } = useWindowMaximised();

  return (
    <header className="title-bar" aria-label="Window title bar">
      <div className="title-bar-leading">
        <img className="title-bar-icon" src={appIcon} alt="" />
        <span className="title-bar-title">{title}</span>
      </div>

      <div className="title-bar-controls">
        <button
          className="title-bar-control"
          onClick={() => void Window.Minimise()}
          title="Minimise"
          type="button"
        >
          <Minus aria-hidden="true" />
        </button>
        <button
          className="title-bar-control title-bar-control-window"
          onClick={() => void toggleMaximised()}
          title={isMaximised ? 'Restore' : 'Maximise'}
          type="button"
        >
          {isMaximised ? <Copy aria-hidden="true" /> : <Square aria-hidden="true" />}
        </button>
        <button
          className="title-bar-control title-bar-control-close"
          onClick={() => void Window.Close()}
          title="Close"
          type="button"
        >
          <X aria-hidden="true" />
        </button>
      </div>
    </header>
  );
};
