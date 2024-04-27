import Editor, { loader } from '@monaco-editor/react';
import { Checkbox, Flex, Spin, Typography } from 'antd';
import type { JSONSchema4 } from 'json-schema';
import * as monaco from 'monaco-editor';
import { configureMonacoYaml } from 'monaco-yaml';
import React, { FC, useEffect, useRef } from 'react';
import yaml from 'yaml';

import styles from './yaml-editor.module.less';

loader.config({ monaco });

export interface YamlEditorProps {
  value: string;
  disabled?: boolean;
  onChange?(value: string | undefined): void;
  className?: string;
  width?: string;
  height?: string;
  schema?: JSONSchema4;
  placeholder?: string;
  isLoading?: boolean;
  isHideManagedFieldsDisplayed?: boolean;
  label?: string;
}

const YamlEditor: FC<YamlEditorProps> = (props) => {
  const editorRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null);
  const {
    value,
    disabled,
    onChange,
    className,
    width,
    height,
    schema,
    placeholder,
    isLoading,
    isHideManagedFieldsDisplayed,
    label
  } = props;
  const [hideManagedFields, setHideManagedFields] = React.useState(!!isHideManagedFieldsDisplayed);

  const handleOnChange = (newValue: string | undefined) => {
    onChange?.(newValue);
  };

  useEffect(() => {
    configureMonacoYaml(monaco, {
      enableSchemaRequest: true,
      hover: true,
      completion: true,
      validate: true,
      isKubernetes: true,
      format: true,
      schemas: schema && [
        {
          uri: 'http://myserver/foo-schema.json',
          fileMatch: ['*'],
          schema
        }
      ]
    });
  }, []);

  const handleEditorDidMount = (editor: monaco.editor.IStandaloneCodeEditor) => {
    editorRef.current = editor;
  };

  const filteredValue = React.useMemo(() => {
    if (!hideManagedFields) {
      return value;
    }

    try {
      const data = yaml.parse(value);
      delete data.metadata.managedFields;

      return yaml.stringify(data);
    } catch (err) {
      return value;
    }
  }, [value, hideManagedFields]);

  if (isLoading) {
    return (
      <Spin tip='Loading' size='small'>
        <div className='content py-8' />
      </Spin>
    );
  }

  return (
    <>
      <Flex justify='space-between' className={isHideManagedFieldsDisplayed || label ? 'my-1' : ''}>
        <div>{label}</div>
        {isHideManagedFieldsDisplayed && (
          <label>
            <Checkbox
              className='mr-2'
              checked={hideManagedFields}
              onChange={(e) => setHideManagedFields(e.target.checked)}
            />
            Hide Managed Fields
          </label>
        )}
      </Flex>
      <div
        style={{ border: '1px solid #d9d9d9', height, overflow: 'hidden' }}
        className={className}
      >
        <Editor
          options={{
            readOnly: disabled,
            lineDecorationsWidth: 5,
            lineNumbersMinChars: 0,
            glyphMargin: false,
            folding: false,
            lineNumbers: 'off',
            minimap: {
              enabled: false
            },
            fontSize: 11
          }}
          width={width}
          height={height}
          language='yaml'
          value={filteredValue}
          onChange={handleOnChange}
          onMount={handleEditorDidMount}
        />

        {placeholder && (
          <p
            className={`${styles.placeholderWrapper} font-mono`}
            onClick={() => {
              editorRef.current?.focus?.();
            }}
          >
            {!filteredValue &&
              placeholder
                ?.trim()
                ?.split('\n')
                .map((line, i) => (
                  <React.Fragment key={i}>
                    {line
                      .split('')
                      .map((char, j) =>
                        char === ' ' ? <React.Fragment key={j}>&nbsp;</React.Fragment> : char
                      )}
                    <br />
                  </React.Fragment>
                ))}
          </p>
        )}
      </div>
      {!props.disabled && schema && (
        <div className='mt-1'>
          <Typography.Text type='secondary'>
            Press <strong>ctrl + space</strong> to show suggestions
          </Typography.Text>
        </div>
      )}
    </>
  );
};

export default YamlEditor;
