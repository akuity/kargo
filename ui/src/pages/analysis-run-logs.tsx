import { Breadcrumb } from 'antd';
import { generatePath, useNavigate, useParams } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { AnalysisRunLogs } from '@ui/features/common/analysis-run-logs';

export const AnalysisRunLogsPage = () => {
  const { name, stageName, analysisRunId } = useParams();
  const navigate = useNavigate();

  if (!name || !stageName || !analysisRunId) {
    return <>Not found</>;
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
      />
    </div>
  );
};
