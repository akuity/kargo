import { faChevronRight } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { ConfigProvider, ConfigProviderProps, Layout } from 'antd';
import { PropsWithChildren } from 'react';

// Breadcrumbs in the topbar are separated by a small chevron. Its list item
// is a flex box so the icon centres on the row instead of the text baseline.
const breadcrumbConfig: ConfigProviderProps['breadcrumb'] = {
  separator: <FontAwesomeIcon icon={faChevronRight} className='text-[10px]' />,
  classNames: { separator: 'flex items-center' }
};

export const BaseHeader = ({ children }: PropsWithChildren) => (
  <Layout.Header
    // The breadcrumb's last item is the current page; AntD already sets it in
    // the full text colour, and medium weight sets it apart from the links.
    className='flex items-center justify-between [&_.ant-breadcrumb-item:last-child]:font-medium'
    // A hairline, like the separators in the sidebar rail. Dark mode sets
    // --kargo-topbar-border to the rail's edge colour; light has no rail edge.
    style={{
      borderBottom:
        '1px solid var(--kargo-topbar-border, var(--kargo-color-border-secondary, rgba(0,0,0,.05)))'
    }}
  >
    <ConfigProvider breadcrumb={breadcrumbConfig}>{children}</ConfigProvider>
  </Layout.Header>
);
