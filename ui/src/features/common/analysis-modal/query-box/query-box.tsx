import { Typography } from 'antd';
import classNames from 'classnames';
import React from 'react';

import styles from './query-box.module.less';

const { Paragraph } = Typography;

interface QueryBoxProps {
  className?: string[] | string;
  query: string;
}

export const QueryBox = ({ className, query }: QueryBoxProps) => {
  const queryTextRef = React.useRef<HTMLDivElement>(null);
  const [canExpand, setCanExpand] = React.useState<boolean>(false);
  const [expanded, toggleExpanded] = React.useState<boolean>(false);

  React.useEffect(() => {
    setCanExpand(queryTextRef.current?.offsetHeight !== queryTextRef.current?.scrollHeight);
  }, [queryTextRef]);

  const expandQuery = () => {
    toggleExpanded(true);
    setCanExpand(false);
  };

  return (
    <div
      ref={queryTextRef}
      className={classNames(
        styles['query-box'],
        canExpand && styles['can-expand'],
        expanded && styles['is-expanded'],
        className
      )}
      title={canExpand ? 'Click to expand query' : undefined}
    >
      <pre className={styles.query} onClick={expandQuery} onKeyDown={expandQuery}>
        {query}
      </pre>
      <Paragraph className={styles['query-copy-button']} copyable={{ text: query }} />
    </div>
  );
};
