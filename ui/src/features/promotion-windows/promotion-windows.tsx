import {
  faCalendarDays,
  faList,
  faPlus,
  faQuestionCircle
} from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Popover, Space, Tabs, Typography } from 'antd';
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
      title={
        <Space size={4}>
          Promotion Windows
          <Popover
            content={
              <Flex vertical gap={6} className='max-w-xs text-xs'>
                <Typography.Text>
                  By default, promotions are always open. Windows restrict <strong>when</strong>{' '}
                  they may run, and only for the Stages a window&apos;s selector matches.
                </Typography.Text>
                <Typography.Text>
                  A <strong>Deny</strong> window blocks promotions while it is active. An{' '}
                  <strong>Allow</strong> window works in reverse - promotions to the Stages it
                  matches run only while an Allow is active, so adding one blocks them for the rest
                  of the time.
                </Typography.Text>
                <Typography.Text>
                  If both are active, Deny wins. Disabled windows are ignored, and windows apply
                  equally to automatic, manual and rollback promotions.
                </Typography.Text>
              </Flex>
            }
          >
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      type='inner'
      className='min-h-full'
      extra={
        <Button icon={<FontAwesomeIcon icon={faPlus} size='sm' />} onClick={() => createWindow()}>
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
