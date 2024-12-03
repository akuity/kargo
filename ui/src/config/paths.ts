export const paths = {
  home: '/',
  projects: '/',
  project: '/project/:name',
  projectCredentials: '/project/:name/credentials',
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

  downloads: '/downloads',
  login: '/login',
  tokenRenew: '/token-renew'
};
