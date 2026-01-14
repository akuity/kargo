import { Flex } from 'antd';

import { CredentialsList } from '@ui/features/common/settings/secrets/credentials-list';
import { GenericCredentialsList } from '@ui/features/common/settings/secrets/generic-credentials-list';

export const SharedSecrets = () => {
  return (
    <Flex gap={16} vertical className='min-h-full'>
      <CredentialsList />
      <GenericCredentialsList />
    </Flex>
  );
};
