export const paths = {
  home: '/',
  projects: '/',
  project: '/project/:name',
  projectEvents: '/project/:name/events',
  stage: '/project/:name/stage/:stageName',
  warehouse: '/project/:name/warehouse/:warehouseName/:tab?',
  freight: '/project/:name/freight/:freightName',
  createStage: '/project/:name/create-stage',
  createWarehouse: '/project/:name/create-warehouse',
  projectSettings: '/project/:name/settings',
  user: '/user',
  settings: '/settings',
  analysisRunLogs: '/ext/project/:name/stage/:stageName/analysis-run/:analysisRunId/logs',
  promotionTasks: '/project/:name/promotion-tasks',

  downloads: '/downloads',
  login: '/login',
  tokenRenew: '/token-renew'
};
