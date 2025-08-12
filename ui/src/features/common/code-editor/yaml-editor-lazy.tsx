import Editor, { loader } from '@monaco-editor/react';
import { Flex, Spin, Typography } from 'antd';
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
  label?: string;
  toolbar?: React.ReactNode;
  resourceType?: string;
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
    label,
    resourceType
  } = props;

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
      // @ts-expect-error correct schema
      schemas: schema && [
        {
          uri: `https://raw.githubusercontent.com/akuity/kargo/${__UI_VERSION__ && __UI_VERSION__ !== 'development' ? __UI_VERSION__ : 'main'}/ui/src/gen/schema/${resourceType || 'stages'}.kargo.akuity.io_v1alpha1.json`,
          fileMatch: ['*'],
          schema
        }
      ]
    });
  }, []);

  // Handle readonly field (without onChange)
  const _value = React.useMemo(() => {
    try {
      const data = yaml.parse(value);

      // Hide managedFields
      if (data?.metadata?.managedFields) {
        delete data.metadata.managedFields;

        return yaml.stringify(data);
      }

      return value;
    } catch (_) {
      return value;
    }
  }, [value]);

  const handleEditorDidMount = (editor: monaco.editor.IStandaloneCodeEditor) => {
    editorRef.current = editor;
  };

  if (isLoading) {
    return (
      <Spin tip='Loading' size='small'>
        <div className='content py-8' />
      </Spin>
    );
  }

  return (
    <>
      <Flex align='center' className={label ? 'mb-2 mt-1' : ''} gap={8}>
        <div>{label}</div>
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
          value={_value}
          onChange={handleOnChange}
          onMount={handleEditorDidMount}
        />

        {placeholder && (
          <p
            className={`${styles.placeholderWrapper} font-mono mt-9`}
            onClick={() => {
              editorRef.current?.focus?.();
            }}
          >
            {!value &&
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
