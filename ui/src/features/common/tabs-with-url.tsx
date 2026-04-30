import { Tabs, TabsProps } from 'antd';
import React from 'react';
import { useNavigate, useLocation } from 'react-router-dom';

interface TabsWithUrlProps extends TabsProps {
  paramKey?: string;
}

const TabsWithUrl: React.FC<TabsWithUrlProps> = ({ paramKey = 'tab', items, ...rest }) => {
  const navigate = useNavigate();
  const location = useLocation();

  const queryParams = new URLSearchParams(location.search);
  const activeKey = queryParams.get(paramKey) || items?.[0]?.key;

  const handleTabChange = (key: string) => {
    queryParams.set(paramKey, key);
    navigate({ search: queryParams.toString() });
  };

  return <Tabs activeKey={activeKey} onChange={handleTabChange} items={items} {...rest} />;
};

export default TabsWithUrl;
