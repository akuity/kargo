import { faBoxes, faCode, faListCheck } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Drawer, Tabs, Typography } from 'antd';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { URLStates } from '@ui/features/utils/url-query-state/states';
import { useURLQueryState } from '@ui/features/utils/url-query-state/use-url-query-state';

const Body = () => {
  const [urlQuery, setURLQuery] = useURLQueryState<URLStates['project']>();

  const tab = urlQuery?.tab || 'wizard';

  return (
    <Tabs
      activeKey={tab}
      onChange={(newTab) =>
        setURLQuery({ ...urlQuery, tab: newTab as URLStates['project']['tab'] })
      }
      items={[
        {
          key: 'wizard',
          icon: <FontAwesomeIcon icon={faListCheck} />,
          label: 'Form',
          children: <>Wizard</>
        },
        {
          key: 'yaml',
          icon: <FontAwesomeIcon icon={faCode} />,
          label: 'YAML',
          children: <>YAML</>
        }
      ]}
    />
  );
};

const CreateWarehouse = (props: ModalComponentProps) => {
  return (
    <Drawer
      open={props.visible}
      onClose={props.hide}
      width='50%'
      title={
        <Typography.Title level={1} className='flex items-center !m-0'>
          <FontAwesomeIcon icon={faBoxes} className='mr-2 text-base text-gray-400' />
          Create Warehouse
        </Typography.Title>
      }
    >
      {props.visible && <Body />}
    </Drawer>
  );
};

export default CreateWarehouse;
