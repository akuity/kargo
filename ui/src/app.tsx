import { QueryClientProvider } from '@tanstack/react-query';
import { ConfigProvider } from 'antd';
import { BrowserRouter, Route, Routes } from 'react-router-dom';

import { paths } from './config/paths';
import { queryClient } from './config/query-client';
import { theme } from './config/theme';

export const App = () => (
  <QueryClientProvider client={queryClient}>
    <ConfigProvider theme={theme}>
      <BrowserRouter>
        <Routes>
          <Route path={paths.home} element={<>Test</>} />
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  </QueryClientProvider>
);
