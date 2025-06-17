import { Breadcrumb, Flex } from 'antd';
import { Route, Routes } from 'react-router-dom';

import { BaseHeader } from '@ui/features/common/layout/base-header';

import { useExtensionsContext } from '../extensions-context';

export const AppExtensions = () => {
  const { appSubpages } = useExtensionsContext();

  return (
    <Flex vertical className='min-h-full'>
      <Routes>
        {appSubpages.map((page) => (
          <Route
            key={page.path}
            path={page.path}
            element={
              <>
                <BaseHeader>
                  <Breadcrumb
                    separator='>'
                    items={[
                      {
                        title: page.label
                      }
                    ]}
                  />
                </BaseHeader>
                <page.component />
              </>
            }
          />
        ))}
      </Routes>
    </Flex>
  );
};
