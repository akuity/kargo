import { faCaretDown, faCaretUp } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Flex } from 'antd';
import { useState } from 'react';

import styles from './git-update-editor.module.less';

export const OriginOverride = ({ children }: { children: React.ReactNode }) => {
  const [showOverride, setShowOverride] = useState(false);
  return (
    <>
      <Flex
        className={styles.originOverride}
        align='center'
        onClick={() => setShowOverride(!showOverride)}
      >
        Origin Override{' '}
        <FontAwesomeIcon icon={showOverride ? faCaretUp : faCaretDown} className='ml-2' />
      </Flex>
      {showOverride && children}
    </>
  );
};
