import { Spin } from 'antd';
import { Suspense } from 'react';

export const SuspenseSpin = ({ children }: { children: React.ReactNode }) => {
  return <Suspense fallback={<Spin />}>{children}</Suspense>;
};
