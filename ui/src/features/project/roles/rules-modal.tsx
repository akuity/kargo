import { Modal } from 'antd';

import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import { PolicyRule } from '@ui/gen/k8s.io/api/rbac/v1/generated_pb';

import { RulesTable } from './rules-table';

export const RulesModal = ({
  rules,
  hide,
  ...props
}: { rules: PolicyRule[]; hide: () => void } & ModalComponentProps) => {
  return (
    <Modal
      {...props}
      title='Rules'
      width={700}
      onCancel={() => {
        hide();
      }}
      footer={<></>}
    >
      <RulesTable rules={rules} />
    </Modal>
  );
};
