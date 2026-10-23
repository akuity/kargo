import { faTag } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Cascader } from 'antd';
import { useMemo } from 'react';

import { useListProjects } from '@ui/gen/api/v2/core/core';

type LabelOption = { value: string; label: string; children?: LabelOption[] };

const splitPair = (pair: string): [string, string] => {
  const idx = pair.indexOf('=');
  return [pair.slice(0, idx), pair.slice(idx + 1)];
};

export const ProjectLabelFilter = ({
  value,
  onChange
}: {
  value: string[];
  onChange: (pairs: string[]) => void;
}) => {
  const { data } = useListProjects();

  const projectLabels = useMemo(
    () => (data?.data?.items ?? []).map((project) => project.metadata?.labels ?? {}),
    [data]
  );

  const selectedByKey = useMemo(() => {
    const byKey = new Map<string, Set<string>>();
    for (const pair of value) {
      const [key, val] = splitPair(pair);
      byKey.set(key, (byKey.get(key) ?? new Set()).add(val));
    }
    return byKey;
  }, [value]);

  // Build the "pick a key, then a value" drill-down. Each key's available
  // values are computed from the projects that satisfy every *other* selected
  // label, so picking one label narrows what the remaining labels offer (while
  // a chosen key still shows all its values, so more of them can be added).
  const options = useMemo<LabelOption[]>(() => {
    const matchesSelectionExcept = (labels: Record<string, string>, exceptKey: string): boolean => {
      for (const [key, vals] of selectedByKey) {
        if (key !== exceptKey && !vals.has(labels[key])) {
          return false;
        }
      }
      return true;
    };

    const allKeys = new Set<string>();
    for (const labels of projectLabels) {
      for (const key of Object.keys(labels)) {
        allKeys.add(key);
      }
    }

    const result: LabelOption[] = [];
    for (const key of Array.from(allKeys).sort((a, b) => a.localeCompare(b))) {
      const values = new Set<string>();
      for (const labels of projectLabels) {
        if (key in labels && matchesSelectionExcept(labels, key)) {
          values.add(labels[key]);
        }
      }
      if (values.size === 0) {
        continue;
      }
      result.push({
        value: key,
        label: key,
        children: Array.from(values)
          .sort()
          // The label doubles as the selected tag's text, which must name the
          // key too: a bare "aws" or "core" is ambiguous across keys.
          .map((val) => ({ value: val, label: `${key}: ${val}` }))
      });
    }
    return result;
  }, [projectLabels, selectedByKey]);

  // The Cascader works with [key, value] paths; our state is "key=value".
  const cascaderValue = useMemo(() => value.map(splitPair), [value]);

  // Several values of one key are OR'd (`key in (a,b)`), different keys AND'd -
  // see buildLabelSelector.
  const handleChange = (paths: (string | number)[][]) => {
    onChange(paths.map((path) => `${String(path[0])}=${String(path[1])}`));
  };

  return (
    <Cascader
      multiple
      allowClear
      showSearch
      expandTrigger='hover'
      // Always report full [key, value] leaf paths, even when a key has a single
      // value (the default SHOW_PARENT would collapse it to just the key).
      showCheckedStrategy={Cascader.SHOW_CHILD}
      options={options}
      value={cascaderValue}
      onChange={handleChange}
      maxTagCount='responsive'
      style={{ flex: 1, minWidth: 200 }}
      placeholder={
        <>
          <FontAwesomeIcon icon={faTag} className='mr-2' />
          Filter by labels...
        </>
      }
    />
  );
};
