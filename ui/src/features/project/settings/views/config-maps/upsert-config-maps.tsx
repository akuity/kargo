import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faAsterisk } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
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

export const UpsertConfigMaps = (props: Props) => {
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

  const creation = !configMapYaml;

  const [yaml, setYaml] = useState(stringify(configMapYAMLExample));

  useEffect(() => {
    if (configMapYaml) {
      setYaml(configMapYaml);
    }
  }, [configMapYaml]);

  const createOrUpdateMutation = useMutation(createOrUpdateResource, {
    onSuccess: () => {
      message.success({
        content: `ConfigMap has been ${creation ? 'created' : 'updated'}`
      });

      if (creation) {
        props.onSuccess?.();
        props.hide();
      } else {
        getConfigMapQuery.refetch();
      }
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
      okText={creation ? 'Create' : 'Update'}
      onOk={onSubmit}
      onCancel={props.hide}
      open={props.visible}
      width='612px'
      title={
        <>
          <FontAwesomeIcon icon={faAsterisk} className='mr-2' />
          {creation ? 'Create' : 'Edit'} ConfigMap
        </>
      }
    >
      <YamlEditor
        label='YAML'
        isLoading={getConfigMapQuery.isLoading}
        height='500px'
        value={yaml}
        onChange={(e) => setYaml(e || '')}
        isHideManagedFieldsDisplayed
      />
    </Modal>
  );
};
