import { faPencil, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Input, Tooltip } from 'antd';
import TextArea from 'antd/es/input/TextArea';
import { useEffect, useState } from 'react';

type SecretEditorProps = {
  secret: Record<string, string>;
  onChange: (newSecret: Record<string, string>) => void;
};

export const SecretEditor = (props: SecretEditorProps) => {
  const secretEntries = Object.entries(props.secret);

  const [lockedSecrets, setLockedSecrets] = useState<string[]>([]);

  // if first render of this component has redacted secrets then it is edit mode
  // lock the existing secrets
  useEffect(() => {
    setLockedSecrets(Object.keys(props.secret || {}));
  }, []);

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
          {lockedSecrets.includes(key) && (
            <>
              <Input type='password' disabled value='redacted' />
              <Tooltip title='Edit this secret'>
                <Button
                  type='text'
                  icon={
                    <FontAwesomeIcon
                      icon={faPencil}
                      className='p-5'
                      onClick={() => setLockedSecrets(lockedSecrets.filter((s) => s !== key))}
                    />
                  }
                />
              </Tooltip>
            </>
          )}

          {/* MULTI-LINE SECRET */}
          {!lockedSecrets.includes(key) && (
            <TextArea
              value={value as string}
              placeholder='secret'
              rows={1}
              onChange={(e) => {
                const newValue = e.target.value;

                props.onChange({ ...(props.secret || {}), [key]: newValue });
              }}
            />
          )}

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
