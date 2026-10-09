import enterpriseFeatures from './enterprise-features.json';

// Features are keyed by Docusaurus doc ID (the doc's path relative to docs/,
// minus numeric prefixes and file extension), e.g.
// "user-guide/reference-docs/promotion-steps/jira".

export const isEnterprise = (docId) => {
    return enterpriseFeatures.enterprise.includes(docId);
};

export const isBeta = (docId) => {
    return enterpriseFeatures.beta.includes(docId);
};
