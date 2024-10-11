import { ConnectError } from '@connectrpc/connect';
import { message } from 'antd';

import { Time } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { Stage } from '@ui/gen/v1alpha1/generated_pb';

export const getSeconds = (ts?: Time): number => Number(ts?.seconds) || 0;

export const onError = (err: ConnectError) => {
  message.error(err?.toString());
};

export const isStageControlFlow = (stage: Stage) =>
  (stage?.spec?.promotionTemplate?.spec?.steps?.length || 0) <= 0;
