import { Spin } from 'antd';
import React, { Suspense, FC } from 'react';

import type { YamlEditorProps } from './yaml-editor-lazy';

export const LazyYamlEditor = React.lazy(() => import('./yaml-editor-lazy'));

export const YamlEditor: FC<YamlEditorProps> = (props) => (
  <Suspense
    fallback={
      <Spin tip='Loading' size='small'>
        <div className='content py-8' />
      </Spin>
    }
  >
    <LazyYamlEditor {...props} />
  </Suspense>
);
