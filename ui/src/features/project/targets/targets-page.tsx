import { Breadcrumb, Flex } from 'antd';
import { useParams } from 'react-router-dom';

import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { BaseHeader } from '@ui/features/common/layout/base-header';

import { useProjectBreadcrumbs } from '../project-utils';

import { Targets } from './targets';

// TargetsPage is the project-level Targets page: the project header with a
// breadcrumb ending in Targets, as the Events and Settings pages have, over
// the Targets list.
export const TargetsPage = () => {
  const { name } = useParams();
  const projectBreadcrumbs = useProjectBreadcrumbs();
  useDocumentTitle(['Targets', name]);

  return (
    <Flex vertical className='min-h-full'>
      <BaseHeader>
        <Breadcrumb separator='>' items={[...projectBreadcrumbs, { title: 'Targets' }]} />
      </BaseHeader>
      <Targets />
    </Flex>
  );
};
