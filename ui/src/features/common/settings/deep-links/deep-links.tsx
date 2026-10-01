import { faPlus, faQuestionCircle } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Button, Card, Flex, Popover, Space, Typography } from 'antd';

import { useConfirmModal } from '@ui/features/common/confirm-modal/use-confirm-modal';
import { useModal } from '@ui/features/common/modal/use-modal';
import { DeepLink } from '@ui/gen/api/v2/models';

import { DeepLinkKind, deepLinkKinds } from './deep-link-kinds';
import { DeepLinkModal } from './deep-link-modal';
import { DeepLinksListView } from './deep-links-list-view';

/** Which config resource the links being edited live on. */
export type DeepLinkScope = 'project' | 'cluster';

export type DeepLinkPatch = {
  freightLinks?: DeepLink[];
  stageLinks?: DeepLink[];
};

type DeepLinksCardProps = {
  kind: DeepLinkKind;
  scope: DeepLinkScope;
  links: DeepLink[];
  loading?: boolean;
  /**
   * Set when the stored config could not be read. The list then stands in for
   * a spec that is actually unknown, so saving would overwrite it.
   */
  loadFailed?: boolean;
  onUpdate: (links: DeepLink[]) => Promise<unknown>;
};

const DeepLinksCard = ({
  kind,
  scope,
  links,
  loading,
  loadFailed,
  onUpdate
}: DeepLinksCardProps) => {
  const { title, resource, allResources, shownAt } = deepLinkKinds[kind];

  const createModal = useModal();
  const editModal = useModal();
  const confirm = useConfirmModal();

  const onCreate = () =>
    createModal.show((p) => (
      <DeepLinkModal {...p} kind={kind} onSubmit={(link) => onUpdate([...links, link])} />
    ));

  const onEdit = (index: number) =>
    editModal.show((p) => (
      <DeepLinkModal
        {...p}
        editing
        kind={kind}
        link={links[index]}
        onSubmit={(next) => onUpdate(links.map((link, i) => (i === index ? next : link)))}
      />
    ));

  const onDelete = (index: number) =>
    confirm({
      title: `Delete ${resource} Link`,
      content: (
        <Typography.Text>
          Are you sure you want to delete{' '}
          <Typography.Text strong>{links[index].title}</Typography.Text>?
        </Typography.Text>
      ),
      onOk: () => onUpdate(links.filter((_, i) => i !== index))
    });

  return (
    <Card
      title={
        <Space size={4}>
          {title}
          <Popover
            content={
              <Flex vertical gap={6} className='max-w-xs text-xs'>
                <Typography.Text>
                  Shown in {shownAt}. Each URL is a Go template evaluated against the {resource}{' '}
                  being viewed, so one definition covers {allResources} at once.
                </Typography.Text>
                <Typography.Text>
                  {scope === 'project'
                    ? `These apply to this project only, and are shown alongside any ${title.toLowerCase()} defined for the whole cluster.`
                    : `These apply to every project, and each project may add ${title.toLowerCase()} of its own.`}
                </Typography.Text>
                <Typography.Text>
                  A link whose URL or condition fails to evaluate is dropped, and nothing says so -
                  which is what happens when one of them reaches for a field the {resource} does not
                  have set.
                </Typography.Text>
              </Flex>
            }
          >
            <Typography.Text type='secondary'>
              <FontAwesomeIcon icon={faQuestionCircle} size='xs' />
            </Typography.Text>
          </Popover>
        </Space>
      }
      type='inner'
      className='flex-1'
      extra={
        // the modal captures the link list as it opens. Until the config has
        // loaded (or is known not to exist) that list is empty, and saving
        // would replace everything already stored
        <Button
          icon={<FontAwesomeIcon icon={faPlus} size='sm' />}
          onClick={onCreate}
          disabled={loading || loadFailed}
        >
          New Link
        </Button>
      }
    >
      <DeepLinksListView
        links={links}
        loading={loading}
        emptyText={`No ${title.toLowerCase()} yet - nothing extra appears in ${shownAt}.`}
        onEdit={onEdit}
        onDelete={onDelete}
      />
    </Card>
  );
};

type DeepLinksProps = {
  scope: DeepLinkScope;
  freightLinks: DeepLink[];
  stageLinks: DeepLink[];
  loading?: boolean;
  loadFailed?: boolean;
  onUpdate: (patch: DeepLinkPatch) => Promise<unknown>;
};

export const DeepLinks = ({
  scope,
  freightLinks,
  stageLinks,
  loading,
  loadFailed,
  onUpdate
}: DeepLinksProps) => (
  <Flex vertical gap={16} className='min-h-full'>
    <DeepLinksCard
      kind='freight'
      scope={scope}
      links={freightLinks}
      loading={loading}
      loadFailed={loadFailed}
      onUpdate={(links) => onUpdate({ freightLinks: links })}
    />
    <DeepLinksCard
      kind='stage'
      scope={scope}
      links={stageLinks}
      loading={loading}
      loadFailed={loadFailed}
      onUpdate={(links) => onUpdate({ stageLinks: links })}
    />
  </Flex>
);
