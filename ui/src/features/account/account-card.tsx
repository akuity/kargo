import { faArrowRightFromBracket } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Avatar, Button, Card, Flex, Tag, Typography } from 'antd';

import { useAuthContext } from '@ui/features/auth/context/use-auth-context';
import { asText, getDisplayName } from '@ui/features/auth/jwt-utils';
import { useGetPublicConfig } from '@ui/gen/api/v2/system/system';

import { isBuiltInAdmin, signInMethod } from './claims';

// AccountCard says who is signed in, to which Kargo and how, and signs them
// out. The avatar is the one on the sidebar rail, so the two read as the same
// person: the brand navy with colorText-level white.
export const AccountCard = () => {
  const { JWTInfo, logout } = useAuthContext();
  const { data: config } = useGetPublicConfig();

  const displayName = getDisplayName(JWTInfo);
  const admin = isBuiltInAdmin(JWTInfo, config?.data);
  const email = asText(JWTInfo?.email);
  // The name can fall back to the email, so skip it when it would repeat.
  const showEmail = !!email && email.toLowerCase() !== displayName.toLowerCase();

  return (
    <Card>
      <Flex align='center' gap={16} wrap>
        <Card.Meta
          // AntD pins the avatar to the top; center it on the text beside it,
          // as the sign-out button is.
          className='!m-0 min-w-0 flex-[1_1_320px] items-center'
          avatar={
            <Avatar size={48} className='!bg-[#30476c] text-lg font-semibold !text-white/85'>
              {Array.from(displayName)[0].toUpperCase()}
            </Avatar>
          }
          title={
            <Flex align='baseline' gap={8}>
              <span className='min-w-0 truncate'>{displayName}</span>
              {showEmail && (
                <Typography.Text type='secondary' className='min-w-0 truncate text-sm font-normal'>
                  {email}
                </Typography.Text>
              )}
              {admin && (
                <Tag color='warning' className='!m-0'>
                  Admin account
                </Tag>
              )}
            </Flex>
          }
          description={
            <>
              Signed in to{' '}
              <Typography.Text className='font-mono text-[13px]'>
                {window.location.host}
              </Typography.Text>{' '}
              via {signInMethod(JWTInfo, admin)}
            </>
          }
        />
        <Button icon={<FontAwesomeIcon icon={faArrowRightFromBracket} />} onClick={logout}>
          Log out
        </Button>
      </Flex>
    </Card>
  );
};
