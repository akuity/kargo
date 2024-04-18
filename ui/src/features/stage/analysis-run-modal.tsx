import { useQuery } from '@connectrpc/connect-query';
import { Button, Modal } from 'antd';
import { useParams } from 'react-router-dom';
import yaml from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { ModalProps } from '@ui/features/common/modal/use-modal';
import { getAnalysisRun } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from "@ui/gen/service/v1alpha1/service_pb";

import { LoadingState } from '../common';

type Props = ModalProps & {
  name: string;
};

export const AnalysisRunModal = ({ visible, hide, name }: Props) => {
  const { name: projectName } = useParams();
  const { data, isLoading } = useQuery(getAnalysisRun, {
    namespace: projectName,
    name,
    format: RawFormat.YAML,
  });
  const manifest = new TextDecoder().decode(data?.result?.value ?? new Uint8Array());

  return (
    <Modal
      open={visible}
      onCancel={hide}
      title='AnalysisRun'
      footer={
        <Button type='primary' onClick={hide}>
          Close
        </Button>
      }
      width={700}
    >
      {isLoading ? (
        <LoadingState />
      ) : (
        <YamlEditor value={manifest} height='500px' disabled />
      )}
    </Modal>
  );
};
