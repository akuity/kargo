import { Modal } from 'antd';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';

import { RulesTable } from './rules-table';

export const RulesModal = ({
  name,
  rules,
  hide,
  ...props
}: { rules: PolicyRule[]; name?: string; hide: () => void } & ModalComponentProps) => {
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
