import { Timestamp } from '@bufbuild/protobuf';
import { describe, expect, test } from 'vitest';

import { IntOrString } from '../../src/gen/k8s.io/apimachinery/pkg/util/intstr/generated_pb';

describe('metav1.Time', () => {
  test('fromJson', () => {
    const time = Timestamp.fromJson('2023-05-30T00:00:00Z');
    expect(time).toBeDefined();
    expect(time.seconds).toBe(BigInt(1685404800));
    expect(time.nanos).toBe(0);
  });

  test('toJson', () => {
    const time = new Timestamp({seconds: BigInt(1685404800), nanos: 0});
    expect(time.toJson()).toBe('2023-05-30T00:00:00Z');
    expect(time.toJsonString()).toBe('"2023-05-30T00:00:00Z"');
  });

  test('toDate', () => {
    const date = new Date('2023-05-30T00:00:00Z');
    const time = new Timestamp({seconds: BigInt(1685404800), nanos: 0});
    expect(time.toDate().getTime()).toBe(date.getTime());
  });
});

describe('intstr.IntOrString', () => {
  test('fromJson - string', () => {
    const intOrString = IntOrString.fromJson('32');
    expect(intOrString).toBeDefined();
    expect(intOrString.type).toBe(IntOrString.TYPE_STRING);
    expect(intOrString.strVal).toBe('32');
  });
  test('fromJson - integer', () => {
    const intOrString = IntOrString.fromJson(32);
    expect(intOrString).toBeDefined();
    expect(intOrString.type).toBe(IntOrString.TYPE_INT);
    expect(intOrString.intVal).toBe(32);
  });
  test('fromJson - integer (bigger than Int32)', () => {
    expect(() => IntOrString.fromJson(Number.MAX_SAFE_INTEGER)).toThrow();
  });
  test('fromJson - integer (smaller than Int32)', () => {
    expect(() => IntOrString.fromJson(Number.MIN_SAFE_INTEGER)).toThrow();
  });
  test('toJson - string', () => {
    const intOrString = new IntOrString({
      type: IntOrString.TYPE_STRING,
      strVal: '32'
    });
    expect(intOrString.toJson()).toBe('32');
  });
  test('toJson - integer', () => {
    const intOrString = new IntOrString({
      type: IntOrString.TYPE_INT,
      intVal: 32
    });
    expect(intOrString.toJson()).toBe(32);
  });
  test('toJson - integer (bigger than Int32)', () => {
    const intOrString = new IntOrString({
      type: IntOrString.TYPE_INT,
      intVal: IntOrString.MIN_INT32 - 1
    });
    expect(() => intOrString.toJson()).toThrow();
  });
  test('toJson - integer (smaller than Int32)', () => {
    const intOrString = new IntOrString({
      type: IntOrString.TYPE_INT,
      intVal: IntOrString.MAX_INT32 + 1
    });
    expect(() => intOrString.toJson()).toThrow();
  });
});
