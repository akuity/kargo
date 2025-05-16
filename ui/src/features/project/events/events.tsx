import { useQuery } from '@connectrpc/connect-query';
import {
  faCheckCircle,
  faExclamationCircle,
  faQuestionCircle
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Breadcrumb, Empty, Flex, Tooltip } from 'antd';
import classNames from 'classnames';
import { format } from 'date-fns';
import moment from 'moment';
import { useParams } from 'react-router-dom';

import { BaseHeader } from '@ui/features/common/layout/base-header';
import { listProjectEvents } from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { Event } from '@ui/gen/k8s.io/api/core/v1/generated_pb';
import { Time } from '@ui/gen/k8s.io/apimachinery/pkg/apis/meta/v1/generated_pb';
import { timestampDate } from '@ui/utils/connectrpc-utils';

import { useProjectBreadcrumbs } from '../project-utils';

const EventValue = ({ label, children }: { label: string; children: React.ReactNode }) => {
  return (
    <div className='flex py-2 items-center'>
      <div className='font-medium text-xs text-right w-20 mr-4 text-gray-400'>{label}</div>
      <div>{children}</div>
    </div>
  );
};

const CircleValue = ({
  children,
  className
}: {
  children: React.ReactNode;
  className?: string;
}) => {
  return (
    <div
      className={classNames(
        'rounded-full flex items-center justify-center font-bold text-white bg-gray-400 w-8 h-8',
        className
      )}
    >
      <div>{children}</div>
    </div>
  );
};

const EventStatus = ({ event, className }: { event: Event; className?: string }) => {
  let color = 'bg-gray-500';
  let icon = faQuestionCircle;

  switch (event.type) {
    case 'Normal':
      color = 'bg-green-500';
      icon = faCheckCircle;
      break;
    case 'Warning':
      color = 'bg-yellow-500';
      icon = faExclamationCircle;
      break;
  }

  return (
    <Tooltip title={event.type}>
      <div className={className}>
        <CircleValue className={color}>
          <FontAwesomeIcon icon={icon} style={{ marginTop: '1px' }} />
        </CircleValue>
      </div>
    </Tooltip>
  );
};

const HumanReadableTimestamp = ({ timestamp }: { timestamp?: Time }) => {
  if (!timestamp) {
    return <>Unknown</>;
  }

  const date = timestampDate(timestamp);
  const fullTimestamp = format(date || '', 'MMM do yyyy HH:mm:ss');
  const fromNow = moment(date).fromNow();

  return (
    <div className='flex items-center'>
      {fromNow} <span className='text-xs font-mono text-gray-400 ml-4'>{fullTimestamp}</span>
    </div>
  );
};

const EventRow = ({ event }: { event: Event }) => {
  return (
    <div className='mb-1 flex flex-col'>
      <div className='uppercase text-xs text-gray-400 ml-auto mr-1 mb-1 font-mono'>
        {event.metadata?.name}
      </div>
      <div className='flex items-center p-4 border border-solid border-gray-200 rounded-md mb-4'>
        <div className='flex flex-col mr-10'>
          <EventStatus event={event} className='mb-2' />
          <Tooltip title={`Count: ${event.count || 0}`}>
            <div>
              <CircleValue>{event.count}x</CircleValue>
            </div>
          </Tooltip>
        </div>
        <div className='flex flex-col mr-10'>
          <EventValue label='Message'>{event.message}</EventValue>
          <EventValue label='Reason'>{event.reason}</EventValue>
        </div>
        <div className='flex flex-col'>
          <EventValue label='First Seen'>
            <HumanReadableTimestamp timestamp={event.firstTimestamp} />
          </EventValue>
          <EventValue label='Last Seen'>
            <HumanReadableTimestamp timestamp={event.lastTimestamp} />
          </EventValue>
        </div>
      </div>
    </div>
  );
};

export const Events = () => {
  const { name } = useParams();
  const { data } = useQuery(listProjectEvents, { project: name });
  const projectBreadcrumbs = useProjectBreadcrumbs();

  return (
    <Flex vertical className='min-h-full'>
      <BaseHeader>
        <Breadcrumb
          separator='>'
          items={[
            ...projectBreadcrumbs,
            {
              title: 'Events'
            }
          ]}
        />
      </BaseHeader>
      <div className='px-4 pt-3 pb-10 flex flex-col flex-1'>
        {(data?.events || []).length > 0 ? (
          data?.events.map((event) => <EventRow key={event.metadata?.name} event={event} />)
        ) : (
          <Empty description='No events' className='my-auto pb-20' />
        )}
      </div>
    </Flex>
  );
};
