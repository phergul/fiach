import { HashRouter, Navigate, Route, Routes } from 'react-router-dom';

import { ThemeProvider } from '@components/Common/ThemeProvider/ThemeProvider';
import { ToastProvider } from '@components/Common/Toast/Toast';
import { OptiScalerSessionProvider } from '@components/Games/OptiScaler/OptiScalerSessionProvider/OptiScalerSessionProvider';
import { Layout } from '@components/Layout/Layout';
import { DevLogsWindow } from '@components/Dev/DevLogsWindow/DevLogsWindow';
import { LogsWindow } from '@components/Logs/LogsWindow/LogsWindow';
import {
  GameApply,
  GameDetails,
  GameOptiScaler,
  GameReShade,
  Library,
  Profiles,
  Settings,
} from '@pages';

const providers = [ThemeProvider, ToastProvider, OptiScalerSessionProvider];

const wrapWithProviders = (Component: React.FC) => {
  return providers.reduce(
    (AccumulatedComponent, Provider) => {
      return <Provider>{AccumulatedComponent}</Provider>;
    },
    <Component />,
  );
};

export const App = () => {
  const windowName = new URLSearchParams(window.location.search).get('window');

  if (windowName === 'logs') {
    return wrapWithProviders(LogsWindow);
  }

  if (windowName === 'dev-logs') {
    return wrapWithProviders(DevLogsWindow);
  }

  return wrapWithProviders(() => (
    <HashRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route index element={<Navigate to="/library" replace />} />
          <Route path="library" element={<Library />} />
          <Route path="library/:gameId/apply/:profileId" element={<GameApply />} />
          <Route path="library/:gameId/apply" element={<GameApply />} />
          <Route path="library/:gameId/optiscaler" element={<GameOptiScaler />} />
          <Route path="library/:gameId/reshade" element={<GameReShade />} />
          <Route path="library/:gameId" element={<GameDetails />} />
          <Route path="profiles" element={<Profiles />} />
          <Route path="settings" element={<Settings />} />
          <Route path="*" element={<Navigate to="/library" replace />} />
        </Route>
      </Routes>
    </HashRouter>
  ));
};
