import { faDiagramProject, faWarehouse } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Empty, Flex, Space, Tooltip, Typography } from 'antd';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';

/**
 * Shown in place of the pipeline when a project has neither Warehouses nor
 * Stages -- an otherwise blank canvas that gives no hint of what to do next.
 *
 * Pinned to the top of the (relatively positioned) pipeline container rather
 * than laid out in flow: the toolbar above floats in graph view but takes up
 * space in list view, which would otherwise shift this down when toggling
 * between the two.
 */
export const PipelineEmpty = (props: { project: string }) => (
  <Flex align='center' justify='center' className='absolute top-32 left-0 right-0 px-4'>
    <Empty
      image={Empty.PRESENTED_IMAGE_SIMPLE}
      description={
        <Flex vertical gap={4} className='max-w-[440px] mx-auto'>
          <Typography.Text strong>This project has no pipeline yet</Typography.Text>
          <Typography.Text type='secondary' className='text-xs'>
            Start with a Warehouse to discover artifacts, then add the Stages to promote that
            Freight through.
          </Typography.Text>
        </Flex>
      }
    >
      <Space>
        <Link to={generatePath(paths.createWarehouse, { name: props.project })}>
          <Button type='primary' icon={<FontAwesomeIcon icon={faWarehouse} />}>
            Create Warehouse
          </Button>
        </Link>
        {/* A disabled button emits no pointer events, so the tooltip needs a
            wrapper of its own to hang off -- same reason the Create dropdown
            wraps its disabled Freight label in a span. */}
        <Tooltip title='Create a Warehouse before creating a Stage.'>
          <span>
            <Button icon={<FontAwesomeIcon icon={faDiagramProject} />} disabled>
              Create Stage
            </Button>
          </span>
        </Tooltip>
      </Space>
    </Empty>
  </Flex>
);
