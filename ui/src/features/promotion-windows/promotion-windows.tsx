import { faCalendarDays, faList, faPlus } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Tabs, Typography } from 'antd';
import { useMemo, useState } from 'react';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { PromotionWindow } from '@ui/gen/api/v2/models';

import { PromotionWindowModal } from './promotion-window-modal';
import { PromotionWindowsCalendarView } from './promotion-windows-calendar-view';
import { PromotionWindowsListView } from './promotion-windows-list-view';

type PromotionWindowsProps = {
  scope: 'project' | 'cluster';
  promotionWindows: PromotionWindow[];
  onUpdate: (promotionWindows: PromotionWindow[]) => void;
};

export const PromotionWindows = ({ scope, promotionWindows, onUpdate }: PromotionWindowsProps) => {
  const [tab, setTab] = useState('calendar');

  const promotionWindowNames = useMemo(
    () => promotionWindows?.map((w) => w.name),
    [promotionWindows]
  );

  const createModal = useModal();
  const createWindow = (promotionWindow?: PromotionWindow) =>
    createModal.show((p) => (
      <PromotionWindowModal
        {...p}
        scope={scope}
        promotionWindow={promotionWindow}
        names={promotionWindowNames}
        onSubmit={(promotionWindow) => onUpdate([...promotionWindows, promotionWindow])}
      />
    ));

  const confirm = useConfirmModal();

  const editModal = useModal();

  const onDelete = (promotionWindow: PromotionWindow) =>
    confirm({
      title: 'Delete Promotion Window',
      content: (
        <Typography.Text>
          Are you sure you want to delete promotion window{' '}
          <Typography.Text strong>{promotionWindow.name}</Typography.Text>?
        </Typography.Text>
      ),
      onOk: () => {
        onUpdate(promotionWindows.filter((window) => window.name !== promotionWindow.name));
        editModal.hide();
      }
    });

  const onEditModalShow = (promotionWindow: PromotionWindow) =>
    editModal.show((p) => (
      <PromotionWindowModal
        {...p}
        scope={scope}
        editing
        promotionWindow={promotionWindow}
        names={promotionWindowNames.filter((name) => name !== promotionWindow?.name)}
        onSubmit={(next) =>
          onUpdate(
            promotionWindows.map((window) => (window.name === promotionWindow.name ? next : window))
          )
        }
        onDelete={() => onDelete(promotionWindow)}
      />
    ));

  return (
    <Card
      title='Promotion Windows'
      type='inner'
      className='min-h-full'
      extra={
        <Button
          type='primary'
          icon={<FontAwesomeIcon icon={faPlus} size='sm' />}
          onClick={() => createWindow()}
        >
          New Window
        </Button>
      }
    >
      <Tabs
        activeKey={tab}
        onChange={setTab}
        items={[
          {
            key: 'calendar',
            label: (
              <>
                <FontAwesomeIcon icon={faCalendarDays} className='mr-2' size='sm' />
                Calendar
              </>
            ),
            children: (
              <PromotionWindowsCalendarView
                promotionWindows={promotionWindows}
                onCreate={createWindow}
                onEdit={onEditModalShow}
              />
            )
          },
          {
            key: 'list',
            label: (
              <>
                <FontAwesomeIcon icon={faList} className='mr-2' size='sm' />
                List
              </>
            ),
            children: (
              <PromotionWindowsListView
                scope={scope}
                promotionWindows={promotionWindows}
                onEdit={onEditModalShow}
                onDelete={onDelete}
              />
            )
          }
        ]}
      />
    </Card>
  );
};
