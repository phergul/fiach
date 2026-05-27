import { HashRouter, Navigate, Route, Routes } from 'react-router-dom';

import { ThemeProvider } from '@components/Common/ThemeProvider/ThemeProvider';
import { ToastProvider } from '@components/Common/Toast/Toast';
import { Layout } from '@components/Layout/Layout';
import { LogsWindow } from '@components/Logs/LogsWindow/LogsWindow';
import { GameApply, GameDetails, Library, Profiles, Settings } from '@pages';

const providers = [ThemeProvider, ToastProvider];

const wrapWithProviders = (Component: React.FC) => {
  return providers.reduce((AccumulatedComponent, Provider) => {
    return <Provider>{AccumulatedComponent}</Provider>;
  }, <Component />);
};

export const App = () => {
  const windowName = new URLSearchParams(window.location.search).get('window');

  if (windowName === 'logs') {
    return wrapWithProviders(LogsWindow);
  }

  return wrapWithProviders(() =>
    <HashRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Navigate to="/library" replace />} />
          <Route path="library" element={<Library />} />
          <Route path="library/:gameId/apply/:profileId" element={<GameApply />} />
          <Route path="library/:gameId/apply" element={<GameApply />} />
          <Route path="library/:gameId" element={<GameDetails />} />
          <Route path="profiles" element={<Profiles />} />
          <Route path="settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/library" replace />} />
        </Route>
      </Routes>
    </HashRouter>
  );
};
