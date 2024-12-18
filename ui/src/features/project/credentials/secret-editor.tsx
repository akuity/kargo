import { faPencil, faTrash } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Flex, Input, Tooltip } from 'antd';
import TextArea from 'antd/es/input/TextArea';
import { useEffect, useState } from 'react';

type SecretEditorProps = {
  secret: [string, string][];
  onChange: (newSecret: [string, string][]) => void;
};

export const SecretEditor = (props: SecretEditorProps) => {
  const secretEntries = props.secret;

  const [lockedSecrets, setLockedSecrets] = useState<string[]>([]);

  // if first render of this component has redacted secrets then it is edit mode
  // lock the existing secrets
  useEffect(() => {
    setLockedSecrets(props.secret.map((secret) => secret[0]));
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

              props.onChange(
                secretEntries.map((entry, origIdx) => {
                  if (idx === origIdx) {
                    return [newKey, value];
                  }

                  return entry;
                })
              );
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

                props.onChange(
                  secretEntries.map((entry) => {
                    if (key === entry[0]) {
                      return [key, newValue];
                    }

                    return entry;
                  })
                );
              }}
            />
          )}

          <Button
            icon={<FontAwesomeIcon icon={faTrash} className='p-5' />}
            type='text'
            danger
            onClick={() => {
              props.onChange(secretEntries.filter((_, origIdx) => idx !== origIdx));
            }}
          />
        </Flex>
      ))}
      <Button onClick={() => props.onChange(secretEntries.concat([['', '']]))} className='mt-2'>
        Add k8s Secret
      </Button>
    </>
  );
};
