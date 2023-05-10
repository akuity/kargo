import environmentsData from '../../demo/environments.json';
import promotionPolicyData from '../../demo/promotion-policies.json';
import promotionData from '../../demo/promotions.json';

export interface Environment {
  metadata: any;
  status: any;
  spec: {
    subscriptions: any[];
  } & any;
}

export interface PromotionPolicy {
  metadata: {
    uid: string;
  } & any;
  authorizedPromoters: any;
  enableAutoPromotion: boolean;
  environment: string;
  kind: string;
}

const promotionPoliciesByEnvironment = new Map<string, any>();
(promotionPolicyData?.items || []).forEach((policy) => {
  if (!promotionPoliciesByEnvironment.has(policy.environment)) {
    promotionPoliciesByEnvironment.set(policy.environment, []);
  }
  promotionPoliciesByEnvironment.get(policy.environment).push(policy);
});

const promotionsByEnvironment = new Map<string, any>();
(promotionData?.items || []).forEach((promotion) => {
  if (!promotionsByEnvironment.has(promotion.spec.environment)) {
    promotionsByEnvironment.set(promotion.spec.environment, []);
  }
  promotionsByEnvironment.get(promotion.spec.environment).push(promotion);
});

export const GetEnvironments = (): Promise<Environment[]> => {
  return new Promise((resolve) => {
    resolve((environmentsData?.items as Environment[]) || []);
  });
};

export const GetPromotionPoliciesForEnvironment = (
  environmentName: string
): Promise<PromotionPolicy[]> => {
  return new Promise((resolve) => {
    resolve(promotionPoliciesByEnvironment.get(environmentName));
  });
};

export const GetPromotionsForEnvironment = (environmentName: string) => {
  return new Promise((resolve) => {
    resolve(promotionsByEnvironment.get(environmentName));
  });
};
