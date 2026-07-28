import { useEffect, useState } from 'react';

import { readSSETextStream } from '@ui/features/project/pipelines/watch-utils';
import { getGetAnalysisRunLogsUrl } from '@ui/gen/api/v2/verifications/verifications';

export const useWatchAnalysisRunLogs = (
  project?: string,
  analysisRun?: string,
  filters?: {
    metricName?: string;
    containerName?: string;
  }
) => {
  const [logs, setLogs] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  const metricName = filters?.metricName;
  const containerName = filters?.containerName;

  useEffect(() => {
    if (!project || !analysisRun || !metricName || !containerName) {
      setLogs('');
      return;
    }

    const abort = new AbortController();
    const url = getGetAnalysisRunLogsUrl(project, analysisRun, { metricName, containerName });

    (async () => {
      let logLine = '';
      setLogs('');
      setIsLoading(true);
      setError('');
      try {
        for await (const chunk of readSSETextStream(url, abort.signal)) {
          logLine += chunk;
          setLogs(logLine);
        }
      } catch (err) {
        if (!abort.signal.aborted) {
          setError(err instanceof Error ? err.message : String(err));
          setLogs('');
        }
      } finally {
        setIsLoading(false);
      }
    })();

    return () => abort.abort();
  }, [project, analysisRun, metricName, containerName]);

  return { logs, isLoading, error };
};
