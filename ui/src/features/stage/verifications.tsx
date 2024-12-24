import { Checkbox, Flex, Table, Tooltip } from 'antd';
import Link from 'antd/es/typography/Link';
import { format } from 'date-fns';
import moment from 'moment';
import { useMemo, useState } from 'react';

import { VerificationInfo } from '@ui/gen/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { AnalysisModal } from '../common/analysis-modal/analysis-modal';
import { useModal } from '../common/modal/use-modal';

import { VerificationIcon } from './verification-icon';

type Props = {
  verifications: VerificationInfo[];
  images: string[];
};

export const Verifications = ({ verifications, images }: Props) => {
  const { show } = useModal();

  const [showImplicitVerifications, setShowImplicitVerifications] = useState(false);

  // non-rollout verifications are now included in specs
  const filteredVerifications = !showImplicitVerifications
    ? verifications.filter((verfication) => verfication.id !== '')
    : verifications;

  const hasImplicitVerifications = useMemo(
    () => verifications.some((v) => v.id === ''),
    [verifications]
  );

  return (
    <>
      {hasImplicitVerifications && (
        <Flex className='mb-4'>
          <Tooltip
            title={`Implicit verifications are meta info if stage does not have any explicit verifications defined in spec. Kargo fallback stage's health to how last promotion performed.`}
          >
            <Checkbox
              onChange={(e) => setShowImplicitVerifications(e.target.checked)}
              className='ml-auto'
            >
              Show implicit verifications
            </Checkbox>
          </Tooltip>
        </Flex>
      )}

      <Table<(typeof verifications)[number]>
        dataSource={filteredVerifications}
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
            const date = timestampDate(verification.startTime);
            return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
          }}
        />
        <Table.Column<(typeof verifications)[number]>
          title='Duration'
          render={(_, verification) => {
            try {
              const startTime = timestampDate(verification.startTime);
              const finishTime = timestampDate(verification.finishTime);

              const timeTook = moment.duration(moment(finishTime).diff(moment(startTime)));

              return timeTook.humanize();
            } catch {
              return null;
            }
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
                  <AnalysisModal
                    {...p}
                    analysisName={val.analysisRun?.name || ''}
                    images={images}
                  />
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
    </>
  );
};
