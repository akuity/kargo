import { TransportProvider } from '@bufbuild/connect-query';
import { QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider } from 'antd';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { transport } from '@ui/config/transport';
import { Project } from '@ui/pages/project';

import { paths } from './config/paths';
import { queryClient } from './config/query-client';
import { theme } from './config/theme';
import { MainLayout } from './features/common/layout/main-layout';
import { Projects } from './pages/projects';

import './app.less';
import 'antd/dist/reset.css';

export const App = () => (
  <TransportProvider transport={transport}>
    <QueryClientProvider client={queryClient}>
      <ConfigProvider theme={theme}>
        <BrowserRouter>
          <Routes>
            <Route element={<MainLayout />}>
              <Route path={paths.projects} element={<Projects />} />
              <Route path={paths.project} element={<Project />} />
              <Route path={paths.stage} element={<Project />} />
            </Route>
          </Routes>
        </BrowserRouter>
      </ConfigProvider>
    </QueryClientProvider>
  </TransportProvider>
);
