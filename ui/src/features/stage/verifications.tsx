import { Checkbox, Flex, Table, Tooltip } from 'antd';
import Link from 'antd/es/typography/Link';
import { format } from 'date-fns';
import moment from 'moment';
import { useMemo, useState } from 'react';
import { Link as ReactRouterLink } from 'react-router-dom';
import { generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { FreightCollection, VerificationInfo } from '@ui/gen/api/v1alpha1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { AnalysisModal } from '../common/analysis-modal/analysis-modal';
import { useModal } from '../common/modal/use-modal';

import { verificationPhaseIsTerminal } from './utils/verification-phase';
import { VerificationIcon } from './verification-icon';

type Props = {
  verifications: Array<VerificationInfo & { freight: FreightCollection }>;
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
            title={`Implicit verifications occur when a stage with no explicit verification process first reaches a healthy state following a promotion.`}
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
          width={220}
          render={(_, verification) => {
            const date = timestampDate(verification.startTime);
            return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
          }}
        />
        <Table.Column<(typeof verifications)[number]>
          title='Duration'
          render={(_, verification) => {
            if (!verificationPhaseIsTerminal(verification.phase || '')) {
              return null;
            }

            try {
              const startTime = moment(timestampDate(verification.startTime));
              const finishTime = moment(timestampDate(verification.finishTime));

              if (!startTime.isValid() || !finishTime.isValid()) {
                return null;
              }

              const timeTook = moment.duration(finishTime.diff(startTime));

              return timeTook.humanize({ ss: 1 });
            } catch {
              return null;
            }
          }}
        />
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
        <Table.Column<(typeof verifications)[number]>
          title='Freight'
          render={(val, verification) => {
            const freights = verification?.freight?.items;
            const freight = Object.values(freights || {})?.[0];

            if (!verification?.analysisRun?.namespace || !freight?.name) {
              return null;
            }

            return (
              <ReactRouterLink
                to={generatePath(paths.freight, {
                  name: verification?.analysisRun?.namespace,
                  freightName: freight?.name
                })}
              >
                {freight?.name?.slice(0, 7)}
              </ReactRouterLink>
            );
          }}
          width={120}
        />
      </Table>
    </>
  );
};
