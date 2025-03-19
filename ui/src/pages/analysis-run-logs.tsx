import { useQuery } from '@connectrpc/connect-query';
import { Breadcrumb } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { LoadingState } from '@ui/features/common';
import { AnalysisRunLogs } from '@ui/features/common/analysis-run-logs';
import { getAnalysisRun } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { AnalysisRun } from '@ui/gen/api/stubs/rollouts/v1alpha1/generated_pb';

export const AnalysisRunLogsPage = () => {
  const { name, stageName, analysisRunId } = useParams();
  const navigate = useNavigate();

  const getAnalysisRunQuery = useQuery(getAnalysisRun, {
    namespace: name,
    name: analysisRunId
  });

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
        project={name}
        stage={stageName}
        analysisRunId={analysisRunId}
        height='80vh'
        analysisRun={getAnalysisRunQuery?.data?.result?.value as AnalysisRun}
      />
    </div>
  );
};
