import { Timestamp } from '@bufbuild/protobuf';
import { describe, expect, test } from 'vitest';

import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { Promotion, PromotionStatus } from '@ui/gen/v1alpha1/generated_pb';

import { sortPromotions } from './utils/sort';

const newTestPromotion = (phase: string, seconds: number): Promotion => {
  return {
    metadata: {
      creationTimestamp: new Timestamp({
        seconds: BigInt(seconds)
      })
    } as ObjectMeta,
    status: {
      phase
    } as PromotionStatus
  } as Promotion;
};

const running1 = newTestPromotion('Running', 1);
const pending2 = newTestPromotion('Pending', 2);
const running3 = newTestPromotion('Running', 3);
const pending4 = newTestPromotion('Pending', 4);
const succeeded5 = newTestPromotion('Succeeded', 5);
const failed6 = newTestPromotion('Failed', 6);
const unknown7 = newTestPromotion('Unknown', 7);
const running8 = newTestPromotion('Running', 8);
const pending9 = newTestPromotion('Pending', 9);
const running10 = newTestPromotion('Running', 10);
const pending11 = newTestPromotion('Pending', 11);
const succeeded12 = newTestPromotion('Succeeded', 12);
const errored13 = newTestPromotion('Errored', 13);
const unknown14 = newTestPromotion('Unknown', 14);

const testPromotions = [
  running1,
  pending2,
  running3,
  pending4,
  succeeded5,
  failed6,
  unknown7,
  running8,
  pending9,
  running10,
  pending11,
  succeeded12,
  errored13,
  unknown14
];

const orderedPromotions = [
  running10,
  running8,
  running3,
  running1,
  pending11,
  pending9,
  pending4,
  pending2,
  unknown14,
  errored13,
  succeeded12,
  unknown7,
  failed6,
  succeeded5
];

describe('sortPromotions', () => {
  test('sorts promotions', () => {
    expect(testPromotions.sort(sortPromotions)).toEqual(orderedPromotions);
  });
});
