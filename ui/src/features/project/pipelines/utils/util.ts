import { ConnectError } from '@connectrpc/connect';
import { message } from 'antd';

import { Time } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

export const getSeconds = (ts?: Time): number => Number(ts?.seconds) || 0;

export const onError = (err: ConnectError) => {
  message.error(err?.toString());
};
