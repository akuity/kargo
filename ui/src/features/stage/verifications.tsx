import { Table, Tooltip } from 'antd';
import Link from 'antd/es/typography/Link';
import { format } from 'date-fns';

import { VerificationInfo } from '@ui/gen/v1alpha1/generated_pb';
import { k8sApiMachineryTimestampDate } from '@ui/utils/connectrpc-extension';

import { AnalysisModal } from '../common/analysis-modal/analysis-modal';
import { useModal } from '../common/modal/use-modal';

import { VerificationIcon } from './verification-icon';

type Props = {
  verifications: VerificationInfo[];
  images: string[];
};

export const Verifications = ({ verifications, images }: Props) => {
  const { show } = useModal();

  return (
    <Table<(typeof verifications)[number]>
      dataSource={verifications}
      size='small'
      pagination={{ hideOnSinglePage: true }}
      rowKey={(p) => p.id || ''}
    >
      <Table.Column<(typeof verifications)[number]>
        width={28}
        render={(_, verification) => (
          <Tooltip
            placement='right'
            overlay={() => (
              <div className='p-1'>
                <div className='font-semibold'>{verification.phase}</div>
                {verification.message && <div className='mt-1'>{verification.message}</div>}
              </div>
            )}
          >
            <div>
              <VerificationIcon phase={verification.phase || ''} />
            </div>
          </Tooltip>
        )}
      />
      <Table.Column<(typeof verifications)[number]>
        title='Date'
        render={(_, verification) => {
          const date = k8sApiMachineryTimestampDate(verification.startTime);
          return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
        }}
      />
      <Table.Column title='ID' dataIndex='id' />
      <Table.Column<(typeof verifications)[number]>
        title='AnalysisRun'
        dataIndex=''
        render={(val, verification) => (
          <Link
            onClick={() => {
              show((p) => (
                <AnalysisModal {...p} analysisName={val.analysisRun?.name || ''} images={images} />
              ));
            }}
          >
            {verification.analysisRun?.name}
          </Link>
        )}
      />
      <Table.Column
        title='Freight'
        dataIndex='freight'
        render={(val) => val?.substring(0, 7)}
        width={120}
      />
    </Table>
  );
};
