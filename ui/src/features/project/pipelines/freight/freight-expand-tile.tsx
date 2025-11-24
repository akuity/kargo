import { faArrowsLeftRightToLine } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Typography } from 'antd';

import { useFreightTimelineControllerContext } from '../context/freight-timeline-controller-context';

type Props = {
  count: number;
};

export const FreightExpandTile = ({ count }: Props) => {
  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  return (
    <div
      className='flex flex-col justify-center h-full px-2 bg-gray-100 rounded-md text-center cursor-pointer border border-solid border-gray-100 hover:border-gray-200'
      onClick={() =>
        freightTimelineControllerContext?.setPreferredFilter({
          ...freightTimelineControllerContext?.preferredFilter,
          hideUnusedFreights: false
        })
      }
    >
      <Typography.Text type='secondary' className='text-xs'>
        <FontAwesomeIcon icon={faArrowsLeftRightToLine} />
        <br />
        {count}x
        <br />
        freights
      </Typography.Text>
    </div>
  );
};
