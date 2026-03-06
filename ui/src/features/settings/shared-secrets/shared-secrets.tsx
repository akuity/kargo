import { Flex, Typography } from 'antd';

import { CredentialsList } from '@ui/features/common/settings/secrets/credentials-list';
import { GenericCredentialsList } from '@ui/features/common/settings/secrets/generic-credentials-list';

export const SharedSecrets = () => {
  return (
    <Flex gap={16} vertical className='min-h-full'>
      <CredentialsList />
      <GenericCredentialsList
        description={
          <>
            These secrets can be accessed using the{' '}
            <Typography.Text code>sharedSecret()</Typography.Text> helper by all projects
          </>
        }
      />
    </Flex>
  );
};
