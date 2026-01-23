import { Flex } from 'antd';
import { useParams } from 'react-router-dom';

import { APITokensList } from '@ui/features/common/settings/access/api-tokens/api-tokens-list';
import { RolesList } from '@ui/features/common/settings/access/roles/roles-list';

export const AccessSettings = () => {
  const { name: projectName } = useParams();

  return (
    <Flex gap={16} vertical className='min-h-full'>
      <RolesList project={projectName} />
      <APITokensList project={projectName} />
    </Flex>
  );
};
