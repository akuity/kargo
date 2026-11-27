import { faBook, faBoxes } from '@fortawesome/free-solid-svg-icons';
import { ReactElement } from 'react';
import { renderToStaticMarkup } from 'react-dom/server';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, test } from 'vitest';

import { NavItem } from './nav-item';

const isCurrent = (at: string, item: ReactElement) =>
  renderToStaticMarkup(<MemoryRouter initialEntries={[at]}>{item}</MemoryRouter>).includes(
    'aria-current="page"'
  );

const projects = (
  <NavItem icon={faBoxes} path='/' activePaths={['/project/:name', '/create-project']}>
    Projects
  </NavItem>
);

const settings = (
  <NavItem icon={faBoxes} path='/settings'>
    Settings
  </NavItem>
);

describe('NavItem', () => {
  test.for([
    ['/', true],
    ['/project/demo', true],
    ['/project/demo/stage/test', true],
    ['/create-project', true],
    ['/settings', false],
    ['/projects', false]
  ] as const)('Projects at %j is current: %j', ([at, expected]) => {
    expect(isCurrent(at, projects)).toBe(expected);
  });

  test.for([
    ['/settings', true],
    ['/settings/access', true],
    ['/settingsx', false],
    ['/', false]
  ] as const)('Settings at %j is current: %j', ([at, expected]) => {
    expect(isCurrent(at, settings)).toBe(expected);
  });

  test('an external link is never current', () => {
    const docs = (
      <NavItem icon={faBook} path='/' external>
        Docs
      </NavItem>
    );
    expect(isCurrent('/', docs)).toBe(false);
  });
});
