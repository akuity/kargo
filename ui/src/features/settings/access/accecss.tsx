import { Flex } from 'antd';

import { APITokensList } from '@ui/features/common/settings/access/api-tokens/api-tokens-list';
import { RolesList } from '@ui/features/common/settings/access/roles/roles-list';

export const AccessSettings = () => {
  return (
    <Flex gap={16} vertical className='min-h-full'>
      <RolesList systemLevel />
      <APITokensList systemLevel />
    </Flex>
  );
};
