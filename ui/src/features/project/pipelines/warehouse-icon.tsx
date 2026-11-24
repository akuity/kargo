import { faWarehouse } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import classNames from 'classnames';
import { CSSProperties } from 'react';

// The icon in the Warehouse's color, mixed toward black on a light background
// and white on a dark one, so that every color in the palette clears 4.5:1
// against its own tile, over a faint fill of that color.
const coloredTileClassName = classNames(
  'bg-[color:color-mix(in_srgb,var(--warehouse-accent)_18%,transparent)]',
  'text-[color:color-mix(in_srgb,var(--warehouse-accent)_55%,black)]',
  'dark:text-[color:color-mix(in_srgb,var(--warehouse-accent)_55%,white)]'
);

// A project with a single Warehouse colors none of them, and a deleted
// Warehouse has no color either, so the tile goes grey.
const neutralTileClassName =
  'bg-[color:var(--kargo-color-fill-secondary)] text-[color:var(--kargo-color-text-secondary)]';

const sizeClassNames = {
  // beside a Warehouse's name in the freight timeline and its filters
  small: 'w-4 h-4 rounded text-[9px]',
  // in the header of a Warehouse node in the pipeline graph
  large: 'w-[26px] h-[26px] rounded-[7px] text-[12px]'
};

type WarehouseIconProps = {
  // the Warehouse's color in the pipeline graph, if it has one
  color?: string;
  size?: keyof typeof sizeClassNames;
  className?: string;
};

// WarehouseIcon is a Warehouse's mark wherever the pipeline shows it: a tile in
// the Warehouse's color. It always sits beside the Warehouse's name, so it is
// hidden from assistive technology.
export const WarehouseIcon = ({ color, size = 'small', className }: WarehouseIconProps) => (
  <span
    aria-hidden
    className={classNames(
      'flex flex-none items-center justify-center',
      sizeClassNames[size],
      color ? coloredTileClassName : neutralTileClassName,
      className
    )}
    style={color ? ({ '--warehouse-accent': color } as CSSProperties) : undefined}
  >
    <FontAwesomeIcon icon={faWarehouse} />
  </span>
);
