import { Timestamp } from '@bufbuild/protobuf';
import { test, describe, expect } from 'vitest';

describe('metav1.Time', () => {
  test('fromJson', () => {
    const time = Timestamp.fromJson('2023-05-30T00:00:00Z');
    expect(time).toBeDefined();
    expect(time.seconds).toBe(BigInt(1685404800));
    expect(time.nanos).toBe(0);
  });

  test('toJson', () => {
    const time = new Timestamp({ seconds: BigInt(1685404800), nanos: 0 });
    expect(time.toJsonString()).toBe('"2023-05-30T00:00:00Z"');
  });

  test('toDate', () => {
    const date = new Date('2023-05-30T00:00:00Z');
    const time = new Timestamp({ seconds: BigInt(1685404800), nanos: 0 });
    expect(time.toDate().getTime()).toBe(date.getTime());
  });
});
