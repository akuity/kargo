import { useMutation, useQuery } from '@connectrpc/connect-query';
import { message, Modal } from 'antd';
import { useEffect, useState } from 'react';
import { stringify } from 'yaml';

import YamlEditor from '@ui/features/common/code-editor/yaml-editor-lazy';
import { ModalComponentProps } from '@ui/features/common/modal/modal-context';
import {
  createOrUpdateResource,
  getConfigMap
} from '@ui/gen/api/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/api/service/v1alpha1/service_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';

import { configMapYAMLExample } from './config-map-yaml-example';

type Props = ModalComponentProps & {
  // if editing then give name of configmap
  editing?: string;
  project: string;
  onSuccess?: () => void;
};

export const UpsertConfigMapsModal = (props: Props) => {
  const getConfigMapQuery = useQuery(
    getConfigMap,
    {
      project: props.project,
      name: props.editing || '',
      format: RawFormat.YAML
    },
    { enabled: !!props.editing }
  );

  const configMapYaml = decodeRawData(getConfigMapQuery.data);

  const [yaml, setYaml] = useState(stringify(configMapYAMLExample(props.project)));

  useEffect(() => {
    if (configMapYaml) {
      setYaml(configMapYaml);
    }
  }, [configMapYaml]);

  const createOrUpdateMutation = useMutation(createOrUpdateResource, {
    onSuccess: () => {
      message.success({
        content: `ConfigMap has been ${!props.editing ? 'created' : 'updated'}`
      });

      props.onSuccess?.();
      props.hide();
    }
  });

  const onSubmit = () => {
    const textEncoder = new TextEncoder();

    createOrUpdateMutation.mutate({
      manifest: textEncoder.encode(yaml)
    });
  };

  return (
    <Modal
      okButtonProps={{
        loading: createOrUpdateMutation.isPending
      }}
      okText={!props.editing ? 'Create' : 'Update'}
      onOk={onSubmit}
      onCancel={props.hide}
      open={props.visible}
      width='612px'
      title={<>{!props.editing ? 'Create' : 'Edit'} ConfigMap</>}
    >
      <YamlEditor
        label='YAML'
        isLoading={getConfigMapQuery.isLoading}
        height='500px'
        value={yaml}
        onChange={(e) => setYaml(e || '')}
      />
    </Modal>
  );
};
