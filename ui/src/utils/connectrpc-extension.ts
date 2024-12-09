import { Timestamp, timestampDate } from '@bufbuild/protobuf/wkt';

import { Time } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';

export const k8sApiMachineryTimestampDate = (t?: Time | PlainMessage<Time> | string) => {
  if (!t) {
    return null;
  }

  if (typeof t === 'string') {
    return new Date(t);
  }

  // apimachinery time is same as google.protobuf.Timestamp
  return timestampDate(t as unknown as Timestamp);
};

export type PlainMessage<T> = Omit<T, '$typeName' | '$unknown'>;

export type PlainMessageRecursive<T> = T extends object
  ? PlainMessage<{ [K in keyof T]: PlainMessageRecursive<T[K]> }>
  : T;

export type PartialRecursive<T> = T extends object
  ? Partial<{ [K in keyof T]: PartialRecursive<T[K]> }>
  : T;
