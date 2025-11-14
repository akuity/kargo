import enterpriseFeatures from './enterprise-features.json';

export const isProfessional = (item) => {
    return enterpriseFeatures.pro.includes(item.label);
};

export const isBeta = (item) => {
    return enterpriseFeatures.beta.includes(item.label);
}
