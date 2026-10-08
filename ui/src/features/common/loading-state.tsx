import { Spin } from 'antd';

export const LoadingState = () => (
  <Spin description='Loading' size='small'>
    <div className='content py-8' />
  </Spin>
);
