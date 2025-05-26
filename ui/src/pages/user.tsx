import { faQuestionCircle, IconDefinition } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Descriptions, Popover, Tag } from 'antd';
import { DescriptionsItemType } from 'antd/es/descriptions';
import { Navigate } from 'react-router-dom';

import { redirectToQueryParam } from '@ui/config/auth';
import { paths } from '@ui/config/paths';
import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { claimsMapping, isJWTDirty } from '@ui/features/auth/jwt-utils';
import { PageTitle } from '@ui/features/common';

export const User = () => {
  const { JWTInfo } = useAuthContext();

  if (isJWTDirty(JWTInfo)) {
    // other pages depend on API failure code to redirect to login
    // since this page doesn't make any API call, this is the only reason to redirect
    return <Navigate to={`${paths.login}?${redirectToQueryParam}=${paths.user}`} />;
  }

  const items: DescriptionsItemType[] = [];

  for (const [key, value] of Object.entries(JWTInfo || {})) {
    items.push({
      label: <User.Label label={key} />,
      children: Array.isArray(value)
        ? value.map((v) => (
            <Tag key={v} className='m-2'>
              {v}
            </Tag>
          ))
        : `${value}`
    });
  }

  return (
    <div className='p-6'>
      <PageTitle title='User' />

      <Descriptions layout='horizontal' column={2} bordered items={items} />
    </div>
  );
};

User.Label = (props: { icon?: IconDefinition; label: string }) => {
  const claimHelpers = claimsMapping[props.label];
  return (
    <>
      {props.icon && <FontAwesomeIcon icon={props.icon} className='mr-2' />} {props.label}{' '}
      {!!claimHelpers && (
        <>
          ({claimHelpers.label}){' '}
          <Popover content={claimHelpers.description}>
            <FontAwesomeIcon icon={faQuestionCircle} className='text-xs' />
          </Popover>
        </>
      )}
    </>
  );
};
