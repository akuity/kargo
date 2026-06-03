import { Promotion } from '@ui/gen/api/v2/models';

export const getPromotionStage = (p: Promotion) => p?.metadata?.ownerReferences?.[0]?.name;
