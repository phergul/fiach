import { HashRouter, Navigate, Route, Routes } from 'react-router-dom';

import { ThemeProvider } from '@components/Common/ThemeProvider/ThemeProvider';
import { ToastProvider } from '@components/Common/Toast/Toast';
import { Layout } from '@components/Layout/Layout';
import { GameDetails, Library, Logs, Profiles, Settings } from '@pages';

const App = () => {
  return (
    <ThemeProvider>
      <ToastProvider>
        <HashRouter>
          <Routes>
            <Route element={<Layout />}>
              <Route index element={<Navigate to="/library" replace />} />
              <Route path="library" element={<Library />} />
              <Route path="library/:gameId" element={<GameDetails />} />
              <Route path="profiles" element={<Profiles />} />
              <Route path="settings" element={<Settings />} />
              <Route path="logs" element={<Logs />} />
              <Route path="*" element={<Navigate to="/library" replace />} />
            </Route>
          </Routes>
        </HashRouter>
      </ToastProvider>
    </ThemeProvider>
  );
};

export default App;
