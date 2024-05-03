import { Timestamp } from '@bufbuild/protobuf';
import { describe, expect, test } from 'vitest';

import { ObjectMeta } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { Promotion, PromotionStatus } from '@ui/gen/v1alpha1/generated_pb';

import { sortPromotions } from './utils/sort';

const newTestPromotion = (phase: string, name: string): Promotion => {
  return {
    metadata: {
      name
    } as ObjectMeta,
    status: {
      phase
    } as PromotionStatus
  } as Promotion;
};

const running1 = newTestPromotion('Running', 'test-a');
const pending2 = newTestPromotion('Pending', 'test-b');
const running3 = newTestPromotion('Running', 'test-c');
const pending4 = newTestPromotion('Pending', 'test-d');
const succeeded5 = newTestPromotion('Succeeded', 'test-e');
const failed6 = newTestPromotion('Failed', 'test-f');
const unknown7 = newTestPromotion('Unknown', 'test-g');
const running8 = newTestPromotion('Running', 'test-h');
const pending9 = newTestPromotion('Pending', 'test-i');
const running10 = newTestPromotion('Running', 'test-j');
const pending11 = newTestPromotion('Pending', 'test-k');
const succeeded12 = newTestPromotion('Succeeded', 'test-l');
const errored13 = newTestPromotion('Errored', 'test-m');
const unknown14 = newTestPromotion('Unknown', 'test-n');

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
