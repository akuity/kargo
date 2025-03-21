import { ConnectError, createClient } from '@connectrpc/connect';
import { faExternalLink, faSearch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Editor } from '@monaco-editor/react';
import { Checkbox, Empty, Input, Select, Skeleton } from 'antd';
import Alert from 'antd/es/alert/Alert';
import { editor, languages } from 'monaco-editor';
import { useEffect, useMemo, useRef, useState } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { transportWithAuth } from '@ui/config/transport';
import { KargoService } from '@ui/gen/api/service/v1alpha1/service_pb';
import { AnalysisRun } from '@ui/gen/api/stubs/rollouts/v1alpha1/generated_pb';

import { verificationPhaseIsTerminal } from '../stage/utils/verification-phase';

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

  const filterableItems = useMemo(() => {
    const logsEligibleMetrics = props.analysisRun?.spec?.metrics?.filter(
      (metric) => !!metric?.provider?.job
    );

    const containerNames: Record<string, string[]> = {};

    for (const logsEligibleMetric of logsEligibleMetrics || []) {
      const containers = logsEligibleMetric?.provider?.job?.spec?.template?.spec?.containers;

      for (const container of containers || []) {
        if (!containerNames[logsEligibleMetric?.name]) {
          containerNames[logsEligibleMetric?.name] = [];
        }

        containerNames[logsEligibleMetric?.name].push(container?.name);
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

  const [selectedJob, setSelectedJob] = useState(
    props.defaultFilters?.selectedJob || filterableItems?.jobNames?.[0]
  );
  const [selectedContainer, setSelectedContainer] = useState(
    props.defaultFilters?.selectedContainer || filterableItems?.containerNames?.[selectedJob]?.[0]
  );

  useEffect(() => {
    if (
      props.defaultFilters?.selectedContainer &&
      filterableItems?.containerNames?.[selectedJob]?.includes(
        props.defaultFilters?.selectedContainer || ''
      )
    ) {
      setSelectedContainer(props.defaultFilters?.selectedContainer);
      return;
    }
    setSelectedContainer(filterableItems?.containerNames?.[selectedJob]?.[0]);
  }, [selectedJob]);

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

    if (!filterableItems?.containerNames?.[selectedJob]?.includes(selectedContainer)) {
      setLogs('');
      return;
    }

    const promiseClient = createClient(KargoService, transportWithAuth);

    const stream = promiseClient.getAnalysisRunLogs({
      namespace: project,
      name: analysisRunId,
      metricName: selectedJob,
      containerName: selectedContainer
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
  }, [selectedJob, selectedContainer, filterableItems, props.analysisRun]);

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
          value={selectedJob}
          className='ml-2 w-1/5'
          options={filterableItems.jobNames.map((job) => ({
            label: job,
            value: job
          }))}
          onChange={setSelectedJob}
        />

        <label className='font-semibold ml-5'>Container: </label>
        <Select
          className='ml-2 w-1/5'
          value={selectedContainer}
          options={filterableItems.containerNames?.[selectedJob]?.map((container) => ({
            label: container,
            value: container
          }))}
          onChange={setSelectedContainer}
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
            })}?job=${selectedJob}&container=${selectedContainer}&search=${search}`}
            className='ml-auto'
            target='_blank'
          >
            <FontAwesomeIcon icon={faExternalLink} /> Full Screen
          </Link>
        )}
      </div>
      {!logsLoading && logs && (
        <Editor
          defaultLanguage='logs'
          theme='logsTheme'
          value={logs}
          height={props.height || '512px'}
          options={{ readOnly: true, lineNumbers: showLineNumbers ? 'on' : 'off' }}
          onMount={(editor) => {
            logsEditor.current = editor;
            editorDecoration.current = editor.createDecorationsCollection([]);

            if (search) {
              const model = editor.getModel();

              if (model) {
                const matches = model.findMatches(search, true, false, false, null, true);

                const decorations = matches.map((match) => ({
                  range: match.range,
                  options: { inlineClassName: 'bg-yellow-300' }
                }));

                editorDecoration.current?.set(decorations);
              }
            }
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
