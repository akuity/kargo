import { Flex, Typography } from 'antd';
import { useParams } from 'react-router-dom';

import { CredentialsList } from '@ui/features/common/settings/secrets/credentials-list';
import { GenericCredentialsList } from '@ui/features/common/settings/secrets/generic-credentials-list';

export const SecretsSettings = () => {
  const { name = '' } = useParams();

  return (
    <Flex gap={16} vertical className='min-h-full'>
      <CredentialsList project={name} />
      <GenericCredentialsList
        project={name}
        description={
          <>
            These secrets can be accessed using the <Typography.Text code>secret()</Typography.Text>{' '}
            helper
          </>
        }
      />
    </Flex>
  );
};
