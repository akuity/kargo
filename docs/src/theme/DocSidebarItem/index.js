import React from 'react';
import DefaultSidebarItem from '@theme-original/DocSidebarItem';

export default function DocSidebarItem(props) {
  const { item } = props;

  const professional = item?.customProps?.professional;
  const beta = item?.customProps?.beta;

  return (
    <div style={{position: 'relative'}}>
      <DefaultSidebarItem {...props} />

      <div style={{position: 'absolute', top: '50%', right: '4px', transform: 'translateY(-50%)'}}>
        {professional && <span className='tag-small professional'></span>}
        {beta && <span className='tag-small beta' style={{marginLeft: '4px'}}></span>}
      </div>
    </div>
  );
}
