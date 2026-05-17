import { faChevronDown, faExternalLink, faLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Dropdown, Flex, Space, Tooltip, Typography } from 'antd';

import { useGetStageLinks } from '@ui/gen/api/v2/core/core';

export const StageDeepLinks = ({
  projectName,
  stageName
}: {
  projectName?: string;
  stageName?: string;
}) => {
  const { data } = useGetStageLinks(projectName || '', stageName || '', {
    query: { enabled: !!projectName && !!stageName }
  });

  const links = data?.data?.links ?? [];

  if (links.length === 0) {
    return null;
  }

  return (
    <Dropdown
      trigger={['hover']}
      menu={{
        style: { maxHeight: '278px', overflowY: 'auto' },
        items: links.map((link, idx) => ({
          key: idx,
          label: (
            <Tooltip title={link.description} placement='left'>
              <Typography.Link href={link.url} target='_blank' rel='noopener noreferrer'>
                <Flex justify='space-between' align='center' gap={16}>
                  {link.title}
                  <FontAwesomeIcon icon={faExternalLink} size='xs' />
                </Flex>
              </Typography.Link>
            </Tooltip>
          )
        }))
      }}
    >
      <Button>
        <Space size={8}>
          <FontAwesomeIcon icon={faLink} />
          Links
          <FontAwesomeIcon icon={faChevronDown} size='xs' />
        </Space>
      </Button>
    </Dropdown>
  );
};
