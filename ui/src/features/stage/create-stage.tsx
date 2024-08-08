import { useMutation } from '@connectrpc/connect-query';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button, Drawer, Flex, Modal, Space, Tabs, Tooltip, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import { useForm } from 'react-hook-form';
import { z } from 'zod';

import { YamlEditor } from '@ui/features/common/code-editor/yaml-editor';
import { FieldContainer } from '@ui/features/common/form/field-container';
import schema from '@ui/gen/schema/stages.kargo.akuity.io_v1alpha1.json';
import { createResource } from '@ui/gen/service/v1alpha1/service-KargoService_connectquery';
import { zodValidators } from '@ui/utils/validators';
import { getStageYAMLExample } from '../project/pipelines/utils/stage-yaml-example';
import { generatePath, useNavigate } from 'react-router-dom';
import { paths } from '@ui/config/paths';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCode, faInfoCircle, faTheaterMasks } from '@fortawesome/free-solid-svg-icons';


const formSchema = z.object({
    value: zodValidators.requiredString
});

export const CreateStage = ({ project }: { project?: string }) => {
    const navigate = useNavigate();
    const close = () => navigate(generatePath(paths.project, { name: project }));

    const { mutateAsync, isPending } = useMutation(createResource, {
        onSuccess: () => close()
    });

    const { control, handleSubmit } = useForm({
        defaultValues: {
            value: getStageYAMLExample(project || '')
        },
        resolver: zodResolver(formSchema)
    });

    const onSubmit = handleSubmit(async (data) => {
        const textEncoder = new TextEncoder();
        await mutateAsync({
            manifest: textEncoder.encode(data.value)
        });
    });

    if (!project) {
        return
    }

    return (
        <Drawer open={!!project} width={'80%'} closable={false} onClose={close}>
            <Flex align='center' className='mb-4'>
                <Typography.Title level={1} className='flex items-center !m-0'>

                    <FontAwesomeIcon icon={faTheaterMasks} className='mr-2 text-base text-gray-400' />
                    Create Stage
                </Typography.Title>
                <Tooltip title='Stage Documentation' placement='bottom' className='ml-3'>
                    <Typography.Link
                        href='https://kargo.akuity.io/quickstart/#the-test-stage'
                        target='_blank'
                    >
                        <FontAwesomeIcon icon={faInfoCircle} />
                    </Typography.Link>
                </Tooltip>
                <Button onClick={close} className='ml-auto'>Cancel</Button>
            </Flex>

            <Tabs>
                <Tabs.TabPane key='1' tab='YAML Editor' icon={<FontAwesomeIcon icon={faCode} />}>
                    <FieldContainer name='value' control={control}>
                        {({ field: { value, onChange } }) => (
                            <YamlEditor
                                value={value}
                                onChange={(e) => onChange(e || '')}
                                height='500px'
                                schema={schema as JSONSchema4}
                                placeholder={getStageYAMLExample(project)}
                                resourceType='stages'
                            />
                        )}
                    </FieldContainer>
                </Tabs.TabPane>
            </Tabs>

            <Button onClick={onSubmit} loading={isPending}>Create</Button>
        </Drawer >
    );
};

export default CreateStage;
