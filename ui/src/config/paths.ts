export const paths = {
  home: '/',
  appExtensions: '/ext',
  projects: '/',
  project: '/project/:name',
  projectEvents: '/project/:name/events',
  stage: '/project/:name/stage/:stageName',
  warehouse: '/project/:name/warehouse/:warehouseName/:tab?',
  promotion: '/project/:name/promotion/:promotionId',
  promote: '/project/:name/promote/freight/:freight/stage/:stage',
  freight: '/project/:name/freight/:freightName',
  createStage: '/project/:name/create-stage',
  createWarehouse: '/project/:name/create-warehouse',
  projectSettings: '/project/:name/settings',
  projectExtensions: '/project/:name/ext',
  user: '/user',
  settings: '/settings',
  analysisRunLogs: '/ext/project/:name/stage/:stageName/analysis-run/:analysisRunId/logs',
  promotionTasks: '/project/:name/promotion-tasks',

  downloads: '/downloads',
  login: '/login',
  tokenRenew: '/token-renew'
};
