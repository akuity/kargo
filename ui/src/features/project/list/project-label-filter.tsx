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
    const byKey = new Map<string, string>();
    for (const pair of value) {
      const [key, val] = splitPair(pair);
      byKey.set(key, val);
    }
    return byKey;
  }, [value]);

  // Build the "pick a key, then a value" drill-down. Each key's available
  // values are computed from the projects that satisfy every *other* selected
  // label, so picking one label narrows what the remaining labels offer (while
  // a chosen key still shows its alternatives so its value can be switched).
  const options = useMemo<LabelOption[]>(() => {
    const matchesSelectionExcept = (labels: Record<string, string>, exceptKey: string): boolean => {
      for (const [key, val] of selectedByKey) {
        if (key !== exceptKey && labels[key] !== val) {
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
          .map((val) => ({ value: val, label: val }))
      });
    }
    return result;
  }, [projectLabels, selectedByKey]);

  // The Cascader works with [key, value] paths; our state is "key=value".
  const cascaderValue = useMemo(() => value.map(splitPair), [value]);

  const handleChange = (paths: (string | number)[][]) => {
    // Allow at most one value per key: when a value is picked for a key that is
    // already selected, the newly chosen one replaces the previous value.
    const previouslySelected = new Set(value);
    const pairByKey = new Map<string, string>();
    for (const path of paths) {
      const key = String(path[0]);
      const pair = `${key}=${String(path[1])}`;
      const existing = pairByKey.get(key);
      // Keep the pair that wasn't already selected (i.e. the fresh pick).
      if (!existing || previouslySelected.has(existing)) {
        pairByKey.set(key, pair);
      }
    }
    onChange(Array.from(pairByKey.values()));
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
