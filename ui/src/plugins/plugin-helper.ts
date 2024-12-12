import { Promotion, PromotionStep } from '@ui/gen/v1alpha1/generated_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

export const getPromotionStepConfig = (step: PromotionStep): Record<string, unknown> =>
  JSON.parse(
    decodeRawData({
      result: {
        case: 'raw',
        value: step?.config?.raw || new Uint8Array()
      }
    })
  );

export const getPromotionState = (promotion: Promotion): Record<string, Record<string, unknown>> =>
  JSON.parse(
    decodeRawData({
      result: {
        case: 'raw',
        value: promotion?.status?.state?.raw || new Uint8Array()
      }
    })
  );

export const getPromotionStepAlias = (promotionStep: PromotionStep, stepIndex: string) =>
  promotionStep?.as || `step-${stepIndex}`;
