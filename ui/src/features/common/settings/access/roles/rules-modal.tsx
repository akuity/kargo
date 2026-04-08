import { Modal } from 'antd';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { V1PolicyRule } from '@ui/gen/api/v2/models';

import { RulesTable } from './rules-table';

export const RulesModal = ({
  name,
  rules,
  hide,
  ...props
}: { rules: V1PolicyRule[]; name?: string; hide: () => void } & ModalComponentProps) => {
  return (
    <Modal
      {...props}
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
