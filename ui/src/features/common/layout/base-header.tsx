import { Layout } from 'antd';
import { PropsWithChildren } from 'react';

export const BaseHeader = ({ children }: PropsWithChildren) => (
  <Layout.Header
    className='flex items-center justify-between'
    style={{ borderBottom: '2px solid var(--kargo-color-border-secondary, rgba(0,0,0,.05))' }}
  >
    {children}
  </Layout.Header>
);
