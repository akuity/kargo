import React from 'react';
import DefaultSidebarItem from '@theme-original/DocSidebarItem';

export default function DocSidebarItem(props) {
  const { item } = props;

  // Beta badges are intentionally shown only on the page itself, not here.
  const pro = item?.customProps?.pro;

  return (
    <div style={{position: 'relative'}}>
      <DefaultSidebarItem {...props} />

      <div style={{position: 'absolute', top: '50%', right: '4px', transform: 'translateY(-50%)'}}>
        {pro && <span className='tag-small professional'></span>}
      </div>
    </div>
  );
}
