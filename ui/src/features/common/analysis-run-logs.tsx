import { faExternalLink, faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Editor } from '@monaco-editor/react';
import { Checkbox, Input, Select } from 'antd';
import { editor, Range } from 'monaco-editor';
import { useRef } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';

export const AnalysisRunLogs = (props: {
  linkFullScreen?: boolean;
  project: string;
  stage: string;
  analysisRunId: string;
  height?: string;
}) => {
  const logsEditor = useRef<editor.IStandaloneCodeEditor>(null);

  return (
    <>
      <div className='mb-5'>
        <Input
          placeholder='Search'
          className='w-1/3'
          prefix={<FontAwesomeIcon icon={faSearch} className='mr-2' />}
          onChange={(e) => {
            const search = e.target.value;

            const model = logsEditor.current?.getModel();

            if (model) {
              const matches = model.findMatches(search, true, false, false, null, true);

              if (matches?.length > 0) {
                logsEditor.current?.setSelection(matches[0].range);
                logsEditor.current?.revealLineInCenter(
                  matches[matches.length - 1].range.startLineNumber
                );
              } else {
                logsEditor.current?.setSelection(new Range(0, 0, 0, 0));
              }
            }
          }}
        />

        <label className='font-semibold ml-5'>Job: </label>
        <Select
          value='All'
          className='ml-2 w-1/5'
          options={[
            {
              label: 'All'
            },
            {
              label: 'api-check'
            },
            {
              label: 'db-check'
            },
            {
              label: 'cache-check'
            }
          ]}
        />

        <label className='font-semibold ml-5'>Container: </label>
        <Select
          className='ml-2 w-1/5'
          value='All'
          options={[
            {
              label: 'All'
            }
          ]}
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
            target='__blank'
          >
            <FontAwesomeIcon icon={faExternalLink} /> Full Screen
          </Link>
        )}
      </div>
      <Editor
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
        }}
      />
    </>
  );
};
