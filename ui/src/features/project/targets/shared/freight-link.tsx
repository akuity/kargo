import { Tooltip } from 'antd';
import classNames from 'classnames';
import { Link, generatePath } from 'react-router-dom';

import { paths } from '@ui/config/paths';

/**
 * FreightLink is a piece of Freight wherever it is mentioned: its short label,
 * its full name on hover, and a link to it. The label is passed in rather than
 * resolved here, because a table of Targets mentions Freight dozens of times
 * over and would otherwise resolve the Project's Freight map once per mention.
 */
export const FreightLink = ({
  project,
  name,
  label,
  className
}: {
  project: string;
  name: string;
  label: string;
  className?: string;
}) => (
  <Tooltip title={name}>
    <Link
      className={classNames('font-mono', className)}
      to={generatePath(paths.freight, { name: project, freightName: name })}
    >
      {label}
    </Link>
  </Tooltip>
);
