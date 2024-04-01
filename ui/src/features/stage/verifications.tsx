import {
  faCircleCheck,
  faCircleExclamation,
  faCircleNotch,
  faCircleQuestion,
  faHourglassStart
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Popover, Table, Tooltip, theme } from 'antd';
import { format } from 'date-fns';
import React from 'react';

import { Stage } from '@ui/gen/v1alpha1/generated_pb';

type Props = {
  stage: Stage;
};

export const Verifications = ({ stage }: Props) => {
  const verifications = React.useMemo(
    () =>
      (stage.status?.history || []).flatMap((freight) =>
        freight.verificationHistory.map((verification) => ({
          freight: freight.name,
          ...verification
        }))
      ),
    [stage]
  );

  return (
    <Table<(typeof verifications)[number]>
      dataSource={verifications}
      size='small'
      pagination={{ hideOnSinglePage: true }}
      rowKey={(p) => p.id || ''}
    >
      <Table.Column<(typeof verifications)[number]>
        width={28}
        render={(_, verification) => {
          switch (verification.phase) {
            case 'Successful':
              return (
                <Popover title={verification.phase} placement='right'>
                  <FontAwesomeIcon
                    color={theme.defaultSeed.colorSuccess}
                    icon={faCircleCheck}
                    size='lg'
                  />
                </Popover>
              );
            case 'Failed':
            case 'Error':
            case 'Aborted':
              return (
                <Popover
                  content={verification.message}
                  title={verification.phase}
                  placement='right'
                >
                  <FontAwesomeIcon
                    color={theme.defaultSeed.colorError}
                    icon={faCircleExclamation}
                    size='lg'
                  />
                </Popover>
              );
            case 'Running':
              return (
                <Tooltip title={verification.phase} placement='right'>
                  <FontAwesomeIcon icon={faCircleNotch} spin size='lg' />
                </Tooltip>
              );
            case 'Pending':
              return (
                <Tooltip title={verification.phase} placement='right'>
                  <FontAwesomeIcon color='#aaa' icon={faHourglassStart} size='lg' />
                </Tooltip>
              );
            default:
              return (
                <Popover
                  title={verification.phase}
                  content={verification.message}
                  placement='right'
                >
                  <FontAwesomeIcon color='#aaa' icon={faCircleQuestion} size='lg' />
                </Popover>
              );
          }
        }}
      />
      <Table.Column<(typeof verifications)[number]>
        title='Date'
        render={(_, verification) => {
          const date = verification.startTime?.toDate();
          return date ? format(date, 'MMM do yyyy HH:mm:ss') : '';
        }}
      />
      <Table.Column title='ID' dataIndex='id' />
      <Table.Column<(typeof verifications)[number]>
        title='AnalysisRun'
        dataIndex=''
        render={(val, verification) => verification.analysisRun?.name}
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
