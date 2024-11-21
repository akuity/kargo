import { faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Input } from 'antd';
import TextArea from 'antd/es/input/TextArea';

type SecretEditorProps = {
  secret: Record<string, string>;
  onChange: (newSecret: Record<string, string>) => void;
};

export const SecretEditor = (props: SecretEditorProps) => {
  const secretEntries = Object.entries(props.secret);

  return (
    <>
      {secretEntries.map(([key, value], idx) => (
        <Flex key={idx} gap={18} className='mb-3'>
          <Input
            className='h-fit'
            value={key}
            onChange={(e) => {
              const newKey = e.target.value;

              const newSecretData: Record<string, string> = { ...(props.secret || {}) };

              delete newSecretData[key];

              newSecretData[newKey] = value as string;

              props.onChange(newSecretData);
            }}
            placeholder='key'
          />
          {/* MULTI-LINE SECRET */}
          <TextArea
            value={value as string}
            placeholder='secret'
            rows={1}
            onChange={(e) => {
              const newValue = e.target.value;

              props.onChange({ ...(props.secret || {}), [key]: newValue });
            }}
          />
          <Button
            icon={<FontAwesomeIcon icon={faTrash} className='p-5' />}
            type='text'
            danger
            onClick={() => {
              const newSecretData: Record<string, string> = { ...props.secret };

              delete newSecretData[key];

              props.onChange(newSecretData);
            }}
          />
        </Flex>
      ))}
      <Button onClick={() => props.onChange({ ...props.secret, '': '' })} className='mt-2'>
        Add k8s Secret
      </Button>
    </>
  );
};
