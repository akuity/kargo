// URL states source of truths
// when using 'useURLQueryState' hook, use these states as required

export type URLStates /* by page */ = {
  project: {
    create: 'warehouse';

    // for warehouse modal
    tab: 'wizard' | 'yaml';
  };
};
