import React from 'react';
import DefaultSidebarItem from '@theme-original/DocSidebarItem';

export default function DocSidebarItem(props) {
  const { item } = props;

  // Beta badges are intentionally shown only on the page itself, not here.
  const enterprise = item?.customProps?.enterprise;

  return (
    <div style={{position: 'relative'}}>
      <DefaultSidebarItem {...props} />

      <div style={{position: 'absolute', top: '50%', right: '4px', transform: 'translateY(-50%)'}}>
        {enterprise && <span className='tag-small enterprise'></span>}
      </div>
    </div>
  );
}
