import { faLink, faUserShield, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Descriptions, Tag, Typography } from 'antd';
import { DescriptionsItemType } from 'antd/es/descriptions';
import { Navigate } from 'react-router-dom';

import { redirectToQueryParam } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { isAdmin, isJWTDirty } from '@ui/features/auth/jwt-utils';
import { PageTitle } from '@ui/features/common';

export const User = () => {
  const { JWTInfo } = useAuthContext();

  if (isJWTDirty(JWTInfo)) {
    // other pages depend on API failure code to redirect to login
    // since this page doesn't make any API call, this is the only reason to redirect
    return <Navigate to={`${paths.login}?${redirectToQueryParam}=${paths.user}`} />;
  }

  const items: DescriptionsItemType[] = [];

  if (isAdmin(JWTInfo)) {
    items.push({
      children: <User.Label icon={faUserShield} label='Admin' />,
      labelStyle: {
        display: 'none'
      }
    });
  }

  items.push({
    label: <User.Label icon={faLink} label='Issuer' />,
    children: JWTInfo?.iss
  });

  if (JWTInfo?.email) {
    items.push({
      label: <User.Label label='Email' />,
      children: JWTInfo.email
    });
  }

  if (JWTInfo?.preferred_username) {
    items.push({
      label: <User.Label label='Username' />,
      children: JWTInfo.preferred_username
    });
  }

  if (JWTInfo && Object.hasOwn(JWTInfo, 'groups')) {
    items.push({
      label: <User.Label label='Groups' />,
      children:
        (JWTInfo.groups?.length || 0) > 0 ? (
          JWTInfo.groups?.map((group) => (
            <Tag key={group} className='m-2'>
              {group}
            </Tag>
          ))
        ) : (
          <Typography.Text type='secondary'>No Groups</Typography.Text>
        )
    });
  }

  return (
    <div className='p-6'>
      <PageTitle title='User' />

      <Descriptions layout='horizontal' column={1} className='w-5/12' bordered items={items} />
    </div>
  );
};

User.Label = (props: { icon?: IconDefinition; label: string }) => (
  <>
    {props.icon && <FontAwesomeIcon icon={props.icon} className='mr-2' />} {props.label}
  </>
);
