import { Col, Row, Typography } from 'antd';
import { PropsWithChildren } from 'react';

import * as styles from './page-title.module.less';

type Props = {
  title: string;
};

export const PageTitle = ({ children, title }: PropsWithChildren<Props>) => (
  <Row justify='space-between' align='top' className={styles.title}>
    <Col>
      <Typography.Title style={{ marginBottom: 0 }}>{title}</Typography.Title>
    </Col>
    <Col>{children}</Col>
  </Row>
);
