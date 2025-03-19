import { faExternalLink, faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Editor } from '@monaco-editor/react';
import { Checkbox, Input, Select } from 'antd';
import { editor, languages } from 'monaco-editor';
import { useEffect, useMemo, useRef } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { AnalysisRun } from '@ui/gen/api/stubs/rollouts/v1alpha1/generated_pb';

export const AnalysisRunLogs = (props: {
  linkFullScreen?: boolean;
  project: string;
  stage: string;
  analysisRunId: string;
  height?: string;
  analysisRun: AnalysisRun;
}) => {
  const logsEditor = useRef<editor.IStandaloneCodeEditor>(null);
  const editorDecoration = useRef<editor.IEditorDecorationsCollection>(null);

  const filterableItems = useMemo(() => {
    const logsEligibleMetrics = props.analysisRun?.spec?.metrics?.filter(
      (metric) => !!metric?.provider?.job
    );

    const containerNames: string[] = [];

    for (const logsEligibleMetric of logsEligibleMetrics || []) {
      const containers = logsEligibleMetric?.provider?.job?.spec?.template?.spec?.containers;

      for (const container of containers || []) {
        containerNames.push(container?.name);
      }
    }

    return {
      jobNames: logsEligibleMetrics?.map((metric) => metric?.name) || [],
      containerNames
    };
  }, [props.analysisRun]);

  useEffect(() => {
    languages.register({ id: 'logs' });

    languages.setMonarchTokensProvider('logs', {
      tokenizer: {
        root: [
          [/\b\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d{1,9})?(Z|[+-]\d{2}:\d{2})?\b/, 'time-format']
        ]
      }
    });

    editor.defineTheme('logsTheme', {
      base: 'vs',
      inherit: true,
      rules: [
        {
          token: 'time-format',
          foreground: '064497'
        }
      ],
      colors: {}
    });
  }, []);

  return (
    <>
      <div className='mb-5'>
        <Input
          placeholder='Search'
          className='w-1/3'
          prefix={<FontAwesomeIcon icon={faSearch} className='mr-2' />}
          onChange={(e) => {
            const search = e.target.value;

            if (!search) {
              editorDecoration.current?.clear();
              return;
            }

            const model = logsEditor.current?.getModel();

            if (model) {
              const matches = model.findMatches(search, true, false, false, null, true);

              const decorations = matches.map((match) => ({
                range: match.range,
                options: { inlineClassName: 'bg-yellow-300' }
              }));

              editorDecoration.current?.set(decorations);
            }
          }}
        />

        <label className='font-semibold ml-5'>Metric: </label>
        <Select
          value='All'
          className='ml-2 w-1/5'
          options={filterableItems.jobNames.map((job) => ({
            label: job
          }))}
        />

        <label className='font-semibold ml-5'>Container: </label>
        <Select
          className='ml-2 w-1/5'
          value='All'
          options={filterableItems.containerNames.map((container) => ({
            label: container
          }))}
        />
      </div>

      <div className='mb-5 flex'>
        <div className='mt-auto space-x-5'>
          <Checkbox>Line numbers</Checkbox>
          <Checkbox>Timestamps</Checkbox>
          <Checkbox>Follow Logs</Checkbox>
        </div>
        {props.linkFullScreen && (
          <Link
            to={generatePath(paths.analysisRunLogs, {
              name: props.project,
              stageName: props.stage,
              analysisRunId: props.analysisRunId
            })}
            className='ml-auto'
            target='_blank'
          >
            <FontAwesomeIcon icon={faExternalLink} /> Full Screen
          </Link>
        )}
      </div>
      <Editor
        defaultLanguage='logs'
        theme='logsTheme'
        value={`2025-03-18T08:34:08.632211717Z [API] Checking API status...
2025-03-18T08:34:13.637596803Z [API] API is responding with 200 OK
2025-03-18T08:34:10.834131760Z [API Latency] Measuring API response time...
2025-03-18T08:34:15.835621595Z [API Latency] Avg response time: 120ms

2025-03-18T08:34:09.114973842Z [Cache] Checking cache availability...
2025-03-18T08:34:14.116927636Z [Cache] Cache is online
2025-03-18T08:34:11.292887593Z [Cache Performance] Measuring cache hit ratio...
2025-03-18T08:34:16.298691595Z [Cache Performance] Cache hit ratio: 95%

2025-03-18T08:34:08.631638883Z [DB] Checking DB connectivity...
2025-03-18T08:34:13.637494386Z [DB] DB connection successful
2025-03-18T08:34:11.052201301Z [DB Metrics] Fetching DB performance stats...
2025-03-18T08:34:16.054146387Z [DB Metrics] Query response time: 15ms

2025-03-18T08:34:08.632211717Z [API] Checking API status...
2025-03-18T08:34:13.637596803Z [API] API is responding with 200 OK
2025-03-18T08:34:10.834131760Z [API Latency] Measuring API response time...
2025-03-18T08:34:15.835621595Z [API Latency] Avg response time: 120ms

2025-03-18T08:34:09.114973842Z [Cache] Checking cache availability...
2025-03-18T08:34:14.116927636Z [Cache] Cache is online
2025-03-18T08:34:11.292887593Z [Cache Performance] Measuring cache hit ratio...
2025-03-18T08:34:16.298691595Z [Cache Performance] Cache hit ratio: 95%

2025-03-18T08:34:08.631638883Z [DB] Checking DB connectivity...
2025-03-18T08:34:13.637494386Z [DB] DB connection successful
2025-03-18T08:34:11.052201301Z [DB Metrics] Fetching DB performance stats...
2025-03-18T08:34:16.054146387Z [DB Metrics] Query response time: 15ms

2025-03-18T08:34:08.632211717Z [API] Checking API status...
2025-03-18T08:34:13.637596803Z [API] API is responding with 200 OK
2025-03-18T08:34:10.834131760Z [API Latency] Measuring API response time...
2025-03-18T08:34:15.835621595Z [API Latency] Avg response time: 120ms

2025-03-18T08:34:09.114973842Z [Cache] Checking cache availability...
2025-03-18T08:34:14.116927636Z [Cache] Cache is online
2025-03-18T08:34:11.292887593Z [Cache Performance] Measuring cache hit ratio...
2025-03-18T08:34:16.298691595Z [Cache Performance] Cache hit ratio: 95%

2025-03-18T08:34:08.631638883Z [DB] Checking DB connectivity...
2025-03-18T08:34:13.637494386Z [DB] DB connection successful
2025-03-18T08:34:11.052201301Z [DB Metrics] Fetching DB performance stats...
2025-03-18T08:34:16.054146387Z [DB Metrics] Query response time: 15ms

2025-03-18T08:34:08.632211717Z [API] Checking API status...
2025-03-18T08:34:13.637596803Z [API] API is responding with 200 OK
2025-03-18T08:34:10.834131760Z [API Latency] Measuring API response time...
2025-03-18T08:34:15.835621595Z [API Latency] Avg response time: 120ms

2025-03-18T08:34:09.114973842Z [Cache] Checking cache availability...
2025-03-18T08:34:14.116927636Z [Cache] Cache is online
2025-03-18T08:34:11.292887593Z [Cache Performance] Measuring cache hit ratio...
2025-03-18T08:34:16.298691595Z [Cache Performance] Cache hit ratio: 95%

2025-03-18T08:34:08.631638883Z [DB] Checking DB connectivity...
2025-03-18T08:34:13.637494386Z [DB] DB connection successful
2025-03-18T08:34:11.052201301Z [DB Metrics] Fetching DB performance stats...
2025-03-18T08:34:16.054146387Z [DB Metrics] Query response time: 15ms

2025-03-18T08:34:08.632211717Z [API] Checking API status...
2025-03-18T08:34:13.637596803Z [API] API is responding with 200 OK
2025-03-18T08:34:10.834131760Z [API Latency] Measuring API response time...
2025-03-18T08:34:15.835621595Z [API Latency] Avg response time: 120ms

2025-03-18T08:34:09.114973842Z [Cache] Checking cache availability...
2025-03-18T08:34:14.116927636Z [Cache] Cache is online
2025-03-18T08:34:11.292887593Z [Cache Performance] Measuring cache hit ratio...
2025-03-18T08:34:16.298691595Z [Cache Performance] Cache hit ratio: 95%

2025-03-18T08:34:08.631638883Z [DB] Checking DB connectivity...
2025-03-18T08:34:13.637494386Z [DB] DB connection successful
2025-03-18T08:34:11.052201301Z [DB Metrics] Fetching DB performance stats...
2025-03-18T08:34:16.054146387Z [DB Metrics] Query response time: 15ms

2025-03-18T08:34:08.632211717Z [API] Checking API status...
2025-03-18T08:34:13.637596803Z [API] API is responding with 200 OK
2025-03-18T08:34:10.834131760Z [API Latency] Measuring API response time...
2025-03-18T08:34:15.835621595Z [API Latency] Avg response time: 120ms

2025-03-18T08:34:09.114973842Z [Cache] Checking cache availability...
2025-03-18T08:34:14.116927636Z [Cache] Cache is online
2025-03-18T08:34:11.292887593Z [Cache Performance] Measuring cache hit ratio...
2025-03-18T08:34:16.298691595Z [Cache Performance] Cache hit ratio: 95%

2025-03-18T08:34:08.631638883Z [DB] Checking DB connectivity...
2025-03-18T08:34:13.637494386Z [DB] DB connection successful
2025-03-18T08:34:11.052201301Z [DB Metrics] Fetching DB performance stats...
2025-03-18T08:34:16.054146387Z [DB Metrics] Query response time: 15ms`}
        height={props.height || '512px'}
        options={{ readOnly: true }}
        onMount={(editor) => {
          logsEditor.current = editor;
          editorDecoration.current = editor.createDecorationsCollection([]);
        }}
      />
    </>
  );
};
