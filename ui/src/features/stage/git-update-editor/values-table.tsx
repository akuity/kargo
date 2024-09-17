import { Flex } from 'antd';

import styles from './git-update-editor.module.less';

export const ValuesTable = ({
  show,
  children,
  label = 'Images'
}: {
  show?: boolean;
  children: React.ReactNode;
  label?: string;
}) => (
  <Flex className={styles.imagesTable}>
    {show ? (
      <div className='w-full'>
        <div className='mb-2 font-semibold text-sm text-gray-500'>{label.toUpperCase()}</div>
        {children}
      </div>
    ) : (
      <div className='mx-auto my-auto font-medium text-gray-400'>
        {label} added using the form to the right will be shown here.
      </div>
    )}
  </Flex>
);
