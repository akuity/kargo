import { Flex, Typography } from 'antd';

import { MIN_FREIGHT_CARD_HEIGHT, emptyMessages, getVariant } from './freight-timeline-empty-utils';

type FreightTimelineEmptyProps = {
  /** true when the active filters, rather than the project, explain the absence */
  filtered: boolean;
  hasWarehouses: boolean;
  promotionMode: boolean;
};

export const FreightTimelineEmpty = (props: FreightTimelineEmptyProps) => (
  <Flex
    align='center'
    justify='center'
    className='px-4 text-center'
    style={{ height: MIN_FREIGHT_CARD_HEIGHT }}
  >
    <Typography.Text type='secondary' className='text-xs'>
      {emptyMessages[getVariant(props)]}
    </Typography.Text>
  </Flex>
);
