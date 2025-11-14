import { Header } from 'antd/es/layout/layout';
import { PropsWithChildren } from 'react';

export const BaseHeader = ({ children }: PropsWithChildren) => (
  <Header
    className='flex items-center justify-between'
    style={{ borderBottom: '2px solid rgba(0,0,0,.05)' }}
  >
    {children}
  </Header>
);
