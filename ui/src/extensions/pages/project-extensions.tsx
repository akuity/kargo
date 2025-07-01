import { Breadcrumb, Flex } from 'antd';
import { Route, Routes } from 'react-router-dom';

import { BaseHeader } from '@ui/features/common/layout/base-header';
import { useProjectBreadcrumbs } from '@ui/features/project/project-utils';

import { useExtensionsContext } from '../extensions-context';

export const ProjectExtensions = () => {
  const { projectSubpages } = useExtensionsContext();
  const projectBreadcrumbs = useProjectBreadcrumbs();

  return (
    <Flex vertical className='min-h-full'>
      <Routes>
        {projectSubpages.map((page) => (
          <Route
            key={page.path}
            path={page.path}
            element={
              <>
                <BaseHeader>
                  <Breadcrumb
                    separator='>'
                    items={[
                      ...projectBreadcrumbs,
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
