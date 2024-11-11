import { useMutation, useQuery } from '@connectrpc/connect-query';
import { faCancel, faPencil, faSave } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import { WarehouseManifestsGen } from '@ui/features/utils/manifest-generator';
import schema from '@ui/gen/schema/warehouses.kargo.akuity.io_v1alpha1.json';
import {
  getWarehouse,
  updateResource
} from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { RawFormat } from '@ui/gen/service/v1alpha1/service_pb';
import { decodeRawData } from '@ui/utils/decode-raw-data';
import { zodValidators } from '@ui/utils/validators';

type Props = {
  projectName?: string;
  warehouseName?: string;
};

const formSchema = z.object({
  value: zodValidators.requiredString
});

export const EditWarehouse = ({ projectName, warehouseName }: Props) => {
  const [editing, setEditing] = useState(false);
  const { data, isLoading } = useQuery(getWarehouse, {
    project: projectName,
    name: warehouseName,
    format: RawFormat.YAML
  });

  const { mutateAsync, isPending } = useMutation(updateResource, {
    onSuccess: () => setEditing(false)
  });

  const { control, handleSubmit } = useForm({
    values: {
      value: decodeRawData(data)
    },
    resolver: zodResolver(formSchema)
  });

  const onSubmit = handleSubmit(async (data) => {
    const textEncoder = new TextEncoder();
    await mutateAsync({
      manifest: textEncoder.encode(data.value)
    });
  });

  return (
    <div>
      <FieldContainer name='value' control={control}>
        {({ field: { value, onChange } }) => (
          <YamlEditor
            value={value}
            onChange={(e) => onChange(e || '')}
            height='500px'
            schema={schema as JSONSchema4}
            placeholder={WarehouseManifestsGen.v1alpha1({
              projectName: projectName || '',
              warehouseName: warehouseName || '',
              spec: {
                subscriptions: [
                  {
                    image: {
                      repoURL: 'public.ecr.aws/nginx/nginx',
                      semverConstraint: '^1.24.0',
                      ignoreTags: []
                    }
                  }
                ]
              }
            })}
            isLoading={isLoading}
            isHideManagedFieldsDisplayed
            disabled={!editing}
            resourceType='warehouses'
            toolbar={
              <>
                {editing && (
                  <Button
                    type='primary'
                    onClick={onSubmit}
                    loading={isPending}
                    className='mr-2'
                    icon={<FontAwesomeIcon icon={faSave} />}
                  >
                    Save
                  </Button>
                )}
                <Button
                  type='default'
                  icon={<FontAwesomeIcon icon={editing ? faCancel : faPencil} size='1x' />}
                  onClick={() => setEditing(!editing)}
                >
                  {editing ? 'Cancel' : 'Edit'}
                </Button>
              </>
            }
          />
        )}
      </FieldContainer>
    </div>
  );
};
