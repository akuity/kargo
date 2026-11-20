import { faMinus, faPlus, faRefresh, faWarehouse } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card } from 'antd';
import classNames from 'classnames';
import { formatDistance } from 'date-fns';
import { CSSProperties, useContext, useMemo } from 'react';
import { generatePath, Link } from 'react-router-dom';

import { paths } from '@ui/config/paths';
import { ColorContext } from '@ui/context/colors';
import { WarehouseExpanded } from '@ui/extend/types';
import { useFreightTimelineControllerContext } from '@ui/features/project/pipelines/context/freight-timeline-controller-context';
import { useRefreshWarehouse } from '@ui/gen/api/v2/core/core';

import { useGraphContext } from '../context/graph-context';

import styles from './node-size-source-of-truth.module.less';
import warehouseStyles from './warehouse-node.module.less';
import { getWarehouseState } from './warehouse-state';

export const WarehouseNode = (props: { warehouse: WarehouseExpanded }) => {
  const colorContext = useContext(ColorContext);

  const freightTimelineControllerContext = useFreightTimelineControllerContext();

  const warehouseName = props.warehouse?.metadata?.name || '';
  const projectName = props.warehouse?.metadata?.namespace || '';

  const state = useMemo(() => getWarehouseState(props.warehouse), [props.warehouse]);

  // No toast on success: the node itself says it is discovering.
  const refreshResourceMutation = useRefreshWarehouse();

  // Taken from the Warehouse alone, not from the request: the refresh
  // annotation arrives by watch a moment after the click, and the node turns
  // to discovering then. Counting the request in as well made the node flicker
  // back to its old state between the request returning and the event landing.
  //
  // The graph holds a quick discovery up for a moment, for the loader and the
  // edge animation alike, so that it does not blink.
  const discovering =
    useGraphContext()?.discoveringWarehouses.has(warehouseName) ?? state.phase === 'discovering';
  const failed = !discovering && state.phase === 'failed';

  const refresh = () =>
    refreshResourceMutation.mutate({ project: projectName, warehouse: warehouseName });

  // A project with a single Warehouse assigns it no color, and the tile goes
  // grey rather than borrowing one.
  const color = colorContext?.warehouseColorMap?.[warehouseName];

  const isSubscriptionHidden =
    freightTimelineControllerContext?.preferredFilter?.hideSubscriptions?.[warehouseName];

  const detailsPath = generatePath(paths.warehouse, { name: projectName, warehouseName });

  return (
    <Card
      size='small'
      // AntD's own card styles match these selectors, so radius and padding
      // are set here rather than lost to a specificity tie in the stylesheet.
      style={{ borderRadius: 10 }}
      styles={{ body: { padding: 0 } }}
      className={classNames(warehouseStyles.warehouseNode, styles['warehouse-node-size'], {
        [warehouseStyles.discovering]: discovering,
        [warehouseStyles.failed]: failed
      })}
    >
      <Button
        size='small'
        style={{ width: 16, height: 16 }}
        aria-label={isSubscriptionHidden ? 'Show subscriptions' : 'Hide subscriptions'}
        icon={<FontAwesomeIcon icon={isSubscriptionHidden ? faPlus : faMinus} />}
        className='absolute left-0 top-[50%] translate-x-[-50%] translate-y-[-50%] text-[8px] z-10'
        onClick={(e) => {
          e.stopPropagation();

          const hiddenSubscriptions = {
            ...freightTimelineControllerContext?.preferredFilter.hideSubscriptions
          };

          if (isSubscriptionHidden) {
            delete hiddenSubscriptions[warehouseName];
          } else {
            hiddenSubscriptions[warehouseName] = true;
          }

          freightTimelineControllerContext?.setPreferredFilter({
            ...freightTimelineControllerContext?.preferredFilter,
            hideSubscriptions: hiddenSubscriptions
          });
        }}
      />

      <div className={warehouseStyles.header}>
        {/* everything up to the refresh button opens the Warehouse */}
        <Link to={detailsPath} className={warehouseStyles.title}>
          <span
            className={classNames(warehouseStyles.tile, !color && warehouseStyles.tileNeutral)}
            style={color ? ({ '--warehouse-accent': color } as CSSProperties) : undefined}
            aria-hidden
          >
            <FontAwesomeIcon icon={faWarehouse} />
          </span>

          <span className={warehouseStyles.name} title={warehouseName}>
            {warehouseName}
          </span>
        </Link>

        <Button
          size='small'
          type='text'
          className={classNames(warehouseStyles.action, 'flex-none')}
          aria-label='Refresh warehouse'
          disabled={discovering}
          icon={<FontAwesomeIcon icon={faRefresh} />}
          onClick={(e) => {
            e.stopPropagation();
            refresh();
          }}
        />
      </div>

      {failed ? (
        <div className={warehouseStyles.error} role='alert'>
          <div className='min-w-0 flex-1'>
            <div className={warehouseStyles.errorTitle}>{state.errorTitle}</div>
            {state.errorMessage && (
              <div className={warehouseStyles.errorMessage} title={state.errorMessage}>
                {state.errorMessage}
              </div>
            )}
          </div>
        </div>
      ) : (
        <div className={warehouseStyles.body}>
          <div className='min-w-0 flex-1'>
            <div className={warehouseStyles.label}>Warehouse</div>
            {/* not a live region: the relative time changes on every render,
                and would be read out each time it did */}
            <div
              className={warehouseStyles.status}
              title={!discovering ? state.discoveredAt?.toString() : undefined}
            >
              {discovering
                ? 'Discovering…'
                : state.discoveredAt
                  ? `Discovered ${formatDistance(state.discoveredAt, new Date(), { addSuffix: true })}`
                  : 'Not yet discovered'}
            </div>
          </div>
        </div>
      )}
    </Card>
  );
};
