import { ConnectError, createClient } from '@connectrpc/connect';
import { faExternalLink, faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Editor } from '@monaco-editor/react';
import { Checkbox, Empty, Input, Select, Skeleton } from 'antd';
import Alert from 'antd/es/alert/Alert';
import { editor } from 'monaco-editor';
import { useEffect, useMemo, useRef, useState } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';
import { KargoService } from '@ui/gen/api/service/v1alpha1/service_pb';
import { AnalysisRun } from '@ui/gen/api/stubs/rollouts/v1alpha1/generated_pb';

import { verificationPhaseIsTerminal } from '../../stage/utils/verification-phase';

import { extractFilters } from './extract-analysis-run';
import {
  monacoEditorLogLanguage,
  monacoEditorLogLanguageTheme,
  useMonacoEditorLogLanguage
} from './use-monaco-editor-log-language';

export const AnalysisRunLogs = (props: {
  linkFullScreen?: boolean;
  height?: string;
  analysisRun: AnalysisRun;
  defaultFilters?: {
    selectedJob?: string;
    selectedContainer?: string;
    search?: string;
  };
}) => {
  const logsEditor = useRef<editor.IStandaloneCodeEditor>(null);
  const editorDecoration = useRef<editor.IEditorDecorationsCollection>(null);

  const filterableItems = useMemo(() => extractFilters(props.analysisRun), [props.analysisRun]);

  useMonacoEditorLogLanguage();

  const [filters, setFilters] = useState(() => {
    const selectedJob = props.defaultFilters?.selectedJob || filterableItems?.jobNames?.[0];

    const selectedContainer =
      props.defaultFilters?.selectedContainer ||
      filterableItems?.containerNames?.[selectedJob]?.[0];

    return { selectedJob, selectedContainer };
  });

  const onSelectJob = (jobName: string) => {
    const containerName = filterableItems?.containerNames?.[jobName]?.[0];

    setFilters({ selectedContainer: containerName, selectedJob: jobName });
  };

  const onSelectContainer = (containerName: string) =>
    setFilters({ ...filters, selectedContainer: containerName });

  const triggerMonacoEditorSearch = (search: string) => {
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
  };

  const [logs, setLogs] = useState('');
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsError, setLogsError] = useState('');

  const project = props.analysisRun?.metadata?.namespace;
  const analysisRunId = props.analysisRun?.metadata?.name;
  const stage = props.analysisRun?.metadata?.labels['kargo.akuity.io/stage'];

  useEffect(() => {
    if (
      !verificationPhaseIsTerminal(props.analysisRun?.status?.phase || '') ||
      !filterableItems?.jobNames?.length
    ) {
      return;
    }

    if (
      !filterableItems?.containerNames?.[filters.selectedJob]?.includes(filters.selectedContainer)
    ) {
      setLogs('');
      return;
    }

    const promiseClient = createClient(KargoService, transportWithAuth);

    const stream = promiseClient.getAnalysisRunLogs({
      namespace: project,
      name: analysisRunId,
      metricName: filters.selectedJob,
      containerName: filters.selectedContainer
    });

    (async () => {
      let logLine = '';
      setLogsLoading(true);
      setLogsError('');
      try {
        for await (const e of stream) {
          logLine += `${e.chunk}`;
          setLogs(logLine);
        }
      } catch (err) {
        if (err instanceof ConnectError) {
          setLogsError(err?.message);
          setLogs('');
        }
      } finally {
        setLogsLoading(false);
      }
    })();
  }, [filters, filterableItems, props.analysisRun]);

  const [showLineNumbers, setShowLineNumbers] = useState(true);
  const [search, setSearch] = useState(props.defaultFilters?.search || '');

  if (!filterableItems?.jobNames?.length) {
    return (
      <Empty description='No job found for this AnalysisRun' image={Empty.PRESENTED_IMAGE_SIMPLE} />
    );
  }

  return (
    <>
      <div className='mb-5'>
        <Input
          placeholder='Search'
          className='w-1/3'
          value={search}
          prefix={<FontAwesomeIcon icon={faSearch} className='mr-2' />}
          onChange={(e) => {
            const search = e.target.value;

            setSearch(search);

            triggerMonacoEditorSearch(search);
          }}
        />

        <label className='font-semibold ml-5'>Metric: </label>
        <Select
          value={filters.selectedJob}
          className='ml-2 w-1/5'
          options={filterableItems.jobNames.map((job) => ({
            label: job,
            value: job
          }))}
          onChange={onSelectJob}
        />

        <label className='font-semibold ml-5'>Container: </label>
        <Select
          className='ml-2 w-1/5'
          value={filters.selectedContainer}
          options={filterableItems.containerNames?.[filters.selectedJob]?.map((container) => ({
            label: container,
            value: container
          }))}
          onChange={onSelectContainer}
        />
      </div>

      <div className='mb-5 flex'>
        <div className='mt-auto space-x-5'>
          <Checkbox
            checked={showLineNumbers}
            onChange={(e) => setShowLineNumbers(e.target.checked)}
          >
            Line numbers
          </Checkbox>
        </div>
        {props.linkFullScreen && (
          <Link
            to={`${generatePath(paths.analysisRunLogs, {
              name: project,
              stageName: stage,
              analysisRunId: analysisRunId
            })}?job=${filters.selectedJob}&container=${filters.selectedContainer}&search=${search}`}
            className='ml-auto'
            target='_blank'
          >
            <FontAwesomeIcon icon={faExternalLink} /> Full Screen
          </Link>
        )}
      </div>
      {!logsLoading && logs && (
        <Editor
          defaultLanguage={monacoEditorLogLanguage}
          theme={monacoEditorLogLanguageTheme}
          value={logs}
          height={props.height || '512px'}
          options={{ readOnly: true, lineNumbers: showLineNumbers ? 'on' : 'off' }}
          onMount={(editor) => {
            logsEditor.current = editor;
            editorDecoration.current = editor.createDecorationsCollection([]);

            triggerMonacoEditorSearch(search);
          }}
        />
      )}
      {!logsLoading && !logs && !logsError && (
        <Empty
          description={`No logs found.${!verificationPhaseIsTerminal(props.analysisRun?.status?.phase || '') ? ' They are available only when verification is in terminal state' : ''}`}
          className='p-10'
        />
      )}
      {logsError && <Alert type='error' description={logsError} />}
      {logsLoading && <Skeleton />}
    </>
  );
};
