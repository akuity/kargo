import { Promotion } from '@ui/gen/api/v1alpha1/generated_pb';

export const getPromotionStage = (p: Promotion) => p?.metadata?.ownerReferences?.[0]?.name;
