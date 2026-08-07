import { Breadcrumb } from 'antd';
import { generatePath, useNavigate, useParams, useSearchParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { AnalysisRunLogs } from '@ui/features/common/analysis-run-logs/analysis-run-logs';
import { useDocumentTitle } from '@ui/features/common/document-title/use-document-title';
import { useGetAnalysisRun } from '@ui/gen/api/v2/verifications/verifications';

export const AnalysisRunLogsPage = () => {
  const { name, stageName, analysisRunId } = useParams();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  useDocumentTitle([analysisRunId && `Logs: ${analysisRunId}`, stageName, name]);

  const getAnalysisRunQuery = useGetAnalysisRun(name || '', analysisRunId || '');

  if (!name || !stageName || !analysisRunId) {
    return <>Not found</>;
  }

  if (getAnalysisRunQuery.isLoading) {
    return <LoadingState />;
  }

  return (
    <div className='px-10'>
      <Breadcrumb
        className='my-5 cursor-pointer'
        items={[
          {
            title: name,
            onClick: () => navigate(generatePath(paths.project, { name }))
          },
          {
            title: stageName,
            onClick: () =>
              navigate(
                generatePath(paths.stage, {
                  name,
                  stageName
                })
              )
          },
          {
            title: analysisRunId
          }
        ]}
      />
      <AnalysisRunLogs
        height='80vh'
        analysisRun={getAnalysisRunQuery.data?.data}
        defaultFilters={{
          selectedJob: searchParams.get('job') || '',
          selectedContainer: searchParams.get('container') || '',
          search: searchParams.get('search') || ''
        }}
      />
    </div>
  );
};
