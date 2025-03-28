export const paths = {
  home: '/',
  projects: '/',
  project: '/project/:name',
  projectCredentials: '/project/:name/secrets',
  projectAnalysisTemplates: '/project/:name/analysis-templates',
  projectEvents: '/project/:name/events',
  projectRoles: '/project/:name/roles',
  stage: '/project/:name/stage/:stageName',
  warehouse: '/project/:name/warehouse/:warehouseName/:tab?',
  freight: '/project/:name/freight/:freightName',
  createStage: '/project/:name/create-stage',
  createWarehouse: '/project/:name/create-warehouse',
  user: '/user',
  settings: '/settings/',
  settingsAnalysisTemplates: '/settings/analysis-templates',
  settingsClusterPromotionTasks: '/settings/cluster-promotion-tasks',
  analysisRunLogs: '/ext/project/:name/stage/:stageName/analysis-run/:analysisRunId/logs',
  promotionTasks: '/project/:name/promotion-tasks',

  downloads: '/downloads',
  login: '/login',
  tokenRenew: '/token-renew'
};
