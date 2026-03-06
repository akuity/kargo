import { faCircleDown } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Typography } from 'antd';

import { Platform, Release } from './types';

export const DownloadLink = ({ url, children }: { url: string; children: React.ReactNode }) => (
  <Button
    type='primary'
    href={url}
    target='_blank'
    rel='noreferrer'
    icon={<FontAwesomeIcon icon={faCircleDown} />}
    size='small'
  >
    {children}
  </Button>
);

export const DownloadItem = ({ title, icon, links, release }: Platform & { release?: Release }) => (
  <Card style={{ width: 250 }}>
    <Typography.Title level={5} style={{ textAlign: 'center', marginBottom: 16 }}>
      <FontAwesomeIcon icon={icon} style={{ marginRight: 8 }} />
      {title}
    </Typography.Title>
    <Flex wrap gap='small' justify='center'>
      {links.map((link) => (
        <DownloadLink key={link.title} url={link.getUrl(release)}>
          {link.title}
        </DownloadLink>
      ))}
    </Flex>
  </Card>
);
