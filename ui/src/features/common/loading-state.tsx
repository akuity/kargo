import { Spin } from 'antd';

export const LoadingState = () => (
  <Spin tip='Loading' size='small'>
    <div className='content py-8' />
  </Spin>
);
