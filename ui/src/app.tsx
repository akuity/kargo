import { TransportProvider } from '@connectrpc/connect-query';
import { QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { ConfigProvider } from 'antd';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { transport } from '@ui/config/transport';
import { Project } from '@ui/pages/project';

import { paths } from './config/paths';
import { queryClient } from './config/query-client';
import { themeConfig } from './config/themeConfig';
import { AuthContextProvider } from './features/auth/context/auth-context-provider';
import { ProtectedRoute } from './features/auth/protected-route';
import { TokenRenew } from './features/auth/token-renew';
import { MainLayout } from './features/common/layout/main-layout';
import { AnalysisRunLogsPage } from './pages/analysis-run-logs';
import { Downloads } from './pages/downloads';
import { Login } from './pages/login/login';
import { Projects } from './pages/projects';
import { Settings } from './pages/settings';
import { User } from './pages/user';

import './app.less';
import 'antd/dist/reset.css';

export const App = () => (
  <TransportProvider transport={transport}>
    <QueryClientProvider client={queryClient}>
      <ConfigProvider theme={themeConfig}>
        <AuthContextProvider>
          <BrowserRouter>
            <Routes>
              <Route element={<ProtectedRoute />}>
                <Route element={<MainLayout />}>
                  <Route path={paths.projects} element={<Projects />} />
                  <Route path={paths.project} element={<Project />} />
                  <Route path={paths.projectCredentials} element={<Project tab='credentials' />} />
                  <Route
                    path={paths.projectAnalysisTemplates}
                    element={<Project tab='analysisTemplates' />}
                  />
                  <Route path={paths.projectEvents} element={<Project tab='events' />} />
                  <Route path={paths.projectRoles} element={<Project tab='roles' />} />
                  <Route path={paths.stage} element={<Project />} />
                  <Route path={paths.freight} element={<Project />} />
                  <Route path={paths.warehouse} element={<Project />} />
                  <Route path={paths.downloads} element={<Downloads />} />
                  <Route path={paths.user} element={<User />} />
                  <Route path={paths.settings} element={<Settings />} />
                  <Route
                    path={paths.settingsAnalysisTemplates}
                    element={<Settings section='verification' />}
                  />
                  <Route
                    path={paths.createStage}
                    element={<Project tab='pipelines' creatingStage={true} />}
                  />
                  <Route
                    path={paths.createWarehouse}
                    element={<Project tab='pipelines' creatingWarehouse />}
                  />
                </Route>
                <Route path={paths.analysisRunLogs} element={<AnalysisRunLogsPage />} />
              </Route>
              <Route path={paths.login} element={<Login />} />
              <Route path={paths.tokenRenew} element={<TokenRenew />} />
            </Routes>
          </BrowserRouter>
        </AuthContextProvider>
      </ConfigProvider>
      <ReactQueryDevtools />
    </QueryClientProvider>
  </TransportProvider>
);
