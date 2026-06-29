import React from 'react';
import ReactDOM from 'react-dom/client';

import '@fontsource/atkinson-hyperlegible';

import { App } from '@app';
import { applyThemeCSSVariables } from '@theme/themeCSSVariables';
import { defaultTheme } from '@theme/themes';

import './styles/_theme.scss';
import './styles/_global.scss';

applyThemeCSSVariables(defaultTheme);

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
