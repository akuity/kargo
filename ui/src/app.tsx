import { TransportProvider } from '@bufbuild/connect-query';
import { QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider } from 'antd';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { transport } from '@ui/config/transport';
import { ModalContextProvider } from '@ui/features/common/modal/modal-context';
import { Project } from '@ui/pages/project';

import { paths } from './config/paths';
import { queryClient } from './config/query-client';
import { themeConfig } from './config/themeConfig';
import { AuthContextProvider } from './features/auth/context/auth-context-provider';
import { ProtectedRoute } from './features/auth/protected-route';
import { TokenRenew } from './features/auth/token-renew';
import { MainLayout } from './features/common/layout/main-layout';
import { Login } from './pages/login/login';
import { Projects } from './pages/projects';

import './app.less';
import 'antd/dist/reset.css';

export const App = () => (
  <TransportProvider transport={transport}>
    <QueryClientProvider client={queryClient}>
      <ConfigProvider theme={themeConfig}>
        <ModalContextProvider>
          <AuthContextProvider>
            <BrowserRouter>
              <Routes>
                <Route element={<ProtectedRoute />}>
                  <Route element={<MainLayout />}>
                    <Route path={paths.projects} element={<Projects />} />
                    <Route path={paths.project} element={<Project />} />
                    <Route path={paths.stage} element={<Project />} />
                  </Route>
                </Route>
                <Route path={paths.login} element={<Login />} />
                <Route path={paths.tokenRenew} element={<TokenRenew />} />
              </Routes>
            </BrowserRouter>
          </AuthContextProvider>
        </ModalContextProvider>
      </ConfigProvider>
    </QueryClientProvider>
  </TransportProvider>
);
