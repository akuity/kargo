import { faPencil, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Table, Typography } from 'antd';

import { DeepLink } from '@ui/gen/api/v2/models';

/**
 * Links have no name of their own. Their position in the list is their
 * identity, so each row carries it. The index AntD hands to a column's render
 * counts from the top of the current page, so it can't stand in.
 */
type DeepLinkRow = { link: DeepLink; index: number };

type DeepLinksListViewProps = {
  links: DeepLink[];
  loading?: boolean;
  emptyText: string;
  onEdit: (index: number) => void;
  onDelete: (index: number) => void;
};

export const DeepLinksListView = ({
  links,
  loading,
  emptyText,
  onEdit,
  onDelete
}: DeepLinksListViewProps) => (
  <Table<DeepLinkRow>
    loading={loading}
    rowKey='index'
    dataSource={links.map((link, index) => ({ link, index }))}
    pagination={{ defaultPageSize: 10, hideOnSinglePage: true }}
    size='small'
    scroll={{ x: 'max-content' }}
    locale={{ emptyText }}
    columns={[
      {
        key: 'title',
        title: 'Title',
        render: (_, { link }) => (
          <Flex vertical gap={2}>
            <Typography.Text strong>{link.title}</Typography.Text>
            {link.description && (
              <Typography.Text type='secondary' className='text-xs'>
                {link.description}
              </Typography.Text>
            )}
          </Flex>
        )
      },
      {
        key: 'url',
        title: 'URL',
        render: (_, { link }) => (
          <Typography.Text code className='text-xs'>
            {link.url}
          </Typography.Text>
        )
      },
      {
        key: 'if',
        title: 'Condition',
        render: (_, { link }) =>
          link.if ? (
            <Typography.Text code className='text-xs'>
              {link.if}
            </Typography.Text>
          ) : (
            <Typography.Text type='secondary'>Always shown</Typography.Text>
          )
      },
      {
        key: 'actions',
        render: (_, { index }) => (
          <Flex gap={8} justify='end'>
            <Button
              icon={<FontAwesomeIcon icon={faPencil} size='sm' />}
              onClick={() => onEdit(index)}
              color='default'
              variant='filled'
              size='small'
            >
              Edit
            </Button>
            <Button
              icon={<FontAwesomeIcon icon={faTrash} size='sm' />}
              onClick={() => onDelete(index)}
              color='danger'
              variant='filled'
              size='small'
            >
              Delete
            </Button>
          </Flex>
        )
      }
    ]}
  />
);
