import { faChevronDown, faExternalLink } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Dropdown, Space } from 'antd';

import { useGetArgoCDLinks } from '@ui/features/stage/use-get-argocd-links';
import { ArgoCDShard } from '@ui/gen/api/service/v1alpha1/service_pb';
import { Stage } from '@ui/gen/api/v1alpha1/generated_pb';

type ArgoCDLinkProps = {
  stage: Stage;
  shards?: Record<string, ArgoCDShard>;
};

export const ArgoCDLink = (props: ArgoCDLinkProps) => {
  const shardKey = props.stage?.metadata?.labels['kargo.akuity.io/shard'] || '';
  const argocdShard = props.shards?.[shardKey];

  const argocdLinks = useGetArgoCDLinks(props.stage, argocdShard);

  if (argocdLinks?.length === 1) {
    return (
      <a target='_blank' href={argocdLinks[0]} className='flex items-center'>
        <img src='/argo-logo.svg' alt='Argo' style={{ width: '20px' }} />
        <FontAwesomeIcon icon={faExternalLink} className='text-[8px] ml-1' />
      </a>
    );
  }

  if (argocdLinks?.length > 1) {
    return (
      <Dropdown.Button
        size='small'
        menu={{
          items: argocdLinks.map((link, idx) => {
            const parts = link?.split('/');
            const name = parts?.[parts.length - 1];
            const namespace = parts?.[parts.length - 2];
            return {
              key: idx,
              label: (
                <a target='_blank' href={link}>
                  {namespace} - {name}
                  <FontAwesomeIcon icon={faExternalLink} className='text-xs ml-2' />
                </a>
              )
            };
          })
        }}
        icon={<FontAwesomeIcon icon={faChevronDown} className='text-xs' />}
      >
        <a onClick={(e) => e.preventDefault()}>
          <Space>
            <img src='/argo-logo.svg' alt='Argo' style={{ width: '20px' }} />
          </Space>
        </a>
      </Dropdown.Button>
    );
  }

  if (argocdShard?.url) {
    return (
      <a target='_blank' href={argocdShard.url}>
        <img src='/argo-logo.svg' alt='Argo' style={{ width: '20px' }} />
        <FontAwesomeIcon icon={faExternalLink} className='text-[8px] ml-1' />
      </a>
    );
  }

  return null;
};
