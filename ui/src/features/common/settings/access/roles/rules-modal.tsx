import { Modal } from 'antd';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { V1PolicyRule } from '@ui/gen/api/v2/models';

import { RulesTable } from './rules-table';

export const RulesModal = ({
  name,
  rules,
  hide,
  visible,
  ...props
}: { rules: V1PolicyRule[]; name?: string; hide: () => void } & ModalComponentProps) => {
  return (
    <Modal
      {...props}
      open={visible}
      title={name ? `Rules: ${name}` : 'Rules'}
      width={800}
      onCancel={() => {
        hide();
      }}
      footer={<></>}
    >
      <RulesTable rules={rules} />
    </Modal>
  );
};
