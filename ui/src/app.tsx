import { Project } from '@pages/project';
import { QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider } from 'antd';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { paths } from './config/paths';
import { queryClient } from './config/query-client';
import { theme } from './config/theme';
import { MainLayout } from './features/ui/layout/main-layout';
import { Projects } from './pages/projects';

import './app.less';
import 'antd/dist/reset.css';

export const App = () => (
  <QueryClientProvider client={queryClient}>
    <ConfigProvider theme={theme}>
      <BrowserRouter>
        <Routes>
          <Route element={<MainLayout />}>
            <Route path={paths.projects} element={<Projects />} />
            <Route path={paths.project} element={<Project />} />
            <Route path={paths.environment} element={<Project />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  </QueryClientProvider>
);
