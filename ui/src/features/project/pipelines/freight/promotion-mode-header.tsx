import { faCircleNotch } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Typography } from 'antd';

import { IAction, useActionContext } from '@ui/features/project/pipelines/context/action-context';
import { useDictionaryContext } from '@ui/features/project/pipelines/context/dictionary-context';

export const PromotionModeHeader = (props: { className?: string; loading?: boolean }) => {
  const actionContext = useActionContext();
  const dictionaryContext = useDictionaryContext();

  if (!actionContext?.action) {
    return null;
  }

  if (actionContext?.action?.type === IAction.MANUALLY_APPROVE) {
    return (
      <div className={props.className}>
        <Typography.Text type='secondary'>
          Manually approve freight <b>{actionContext?.action?.freight?.alias}</b>
        </Typography.Text>
        <Button danger type='primary' size='small' onClick={() => actionContext?.cancel()}>
          Cancel
        </Button>
      </div>
    );
  }

  let promotingTo = actionContext?.action?.stage?.metadata?.name || '';

  if (actionContext?.action?.type === IAction.PROMOTE_DOWNSTREAM) {
    promotingTo = [...(dictionaryContext?.subscribersByStage?.[promotingTo] || [])].join(', ');
  }

  return (
    <div className={props.className}>
      <Typography.Text type='secondary'>
        {props.loading && <FontAwesomeIcon icon={faCircleNotch} spin className='mr-2' />}
        Promote to <b>{promotingTo}</b>
      </Typography.Text>
      <Button danger size='small' onClick={() => actionContext.cancel()}>
        Cancel
      </Button>
    </div>
  );
};
