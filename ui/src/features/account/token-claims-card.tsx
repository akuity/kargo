import { faCheck, faCopy } from '@fortawesome/free-solid-svg-icons';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { Card, Flex, Segmented, Table, TableColumnsType, Tag, Typography } from 'antd';
import { useMemo, useState } from 'react';

import { Claim, claimExpired, claimRows, formatClaimTime, relativeTime } from './claims';
import { useNow } from './use-now';

// Relative times read to the minute at best ("less than a minute ago"), so
// refreshing them twice a minute is enough to keep them, and expiry, true.
const CLOCK_INTERVAL_MS = 30_000;

type View = 'readable' | 'json';

// AntD's own copy button, with the icons the rest of the UI copies with. The
// check mark says it worked, so no tooltip; AntD still labels the button
// "Copy" / "Copied" for screen readers.
const copyable = (text: string) => ({
  text,
  tooltips: false,
  icon: [
    <FontAwesomeIcon key='copy' icon={faCopy} />,
    <FontAwesomeIcon key='copied' icon={faCheck} />
  ]
});

// Values are machine-made -- URLs, IDs, group names -- so they're set in
// monospace, and may break anywhere rather than push the table wider.
const valueClass = 'font-mono text-[13px] break-all';

const ClaimValueView = ({ claim, now }: { claim: Claim; now: Date }) => {
  const { value } = claim;

  switch (value.kind) {
    case 'time': {
      const expired = claimExpired(claim, now);
      return (
        <>
          <div className={valueClass}>{formatClaimTime(value.date)}</div>
          <Typography.Text type='secondary' className='text-xs'>
            <Typography.Text type={expired ? 'danger' : 'secondary'} className='text-xs'>
              {expired && 'expired '}
              {relativeTime(value.date, now)}
            </Typography.Text>{' '}
            · Unix {value.seconds}
          </Typography.Text>
        </>
      );
    }
    case 'list':
      return value.items.length ? (
        <Flex gap={4} wrap>
          {value.items.map((item, i) => (
            <Tag key={`${i}-${item}`} className='!m-0 font-mono'>
              {item}
            </Tag>
          ))}
        </Flex>
      ) : (
        <Typography.Text type='secondary'>None</Typography.Text>
      );
    default:
      return value.text ? (
        <span className={valueClass}>{value.text}</span>
      ) : (
        <Typography.Text type='secondary'>Empty</Typography.Text>
      );
  }
};

// TokenClaimsCard lists what the signed-in user's token says about them: each
// claim explained where it is a standard one, or all of it as JSON.
export const TokenClaimsCard = ({ claims }: { claims: Record<string, unknown> }) => {
  const [view, setView] = useState<View>('readable');

  const rows = useMemo(() => claimRows(claims), [claims]);
  const json = useMemo(() => JSON.stringify(claims, null, 2), [claims]);
  const now = useNow(CLOCK_INTERVAL_MS);

  // The table runs edge to edge in the card, so its outer columns take the
  // card's own 24px inset to line up with the title above them.
  const columns: TableColumnsType<Claim> = [
    {
      key: 'label',
      width: 240,
      className: '!pl-6',
      render: (_, claim) => (
        <>
          <Flex align='center' gap={4}>
            <Typography.Text strong>{claim.label}</Typography.Text>
            {claim.label !== claim.key && (
              <Typography.Text code className='text-[11px]'>
                {claim.key}
              </Typography.Text>
            )}
          </Flex>
          {claim.description && (
            <Typography.Text type='secondary' className='block text-xs'>
              {claim.description}
            </Typography.Text>
          )}
        </>
      )
    },
    {
      key: 'value',
      render: (_, claim) => <ClaimValueView claim={claim} now={now} />
    },
    {
      key: 'copy',
      width: 64,
      align: 'right',
      className: '!pr-6',
      render: (_, claim) => <Typography.Text copyable={copyable(claim.copy)} />
    }
  ];

  return (
    <Card
      title={
        <>
          <div>Token claims</div>
          <Typography.Text type='secondary' className='text-xs font-normal'>
            Decoded from the token Kargo signs you in with. Useful when debugging RBAC or SSO
            mappings.
          </Typography.Text>
        </>
      }
      extra={
        <Segmented<View>
          value={view}
          onChange={setView}
          options={[
            { value: 'readable', label: 'Readable' },
            { value: 'json', label: 'JSON' }
          ]}
        />
      }
      styles={{
        header: { paddingBlock: 12 },
        // AntD pulls the body 1px up over the header's rule, which only
        // padding hides; the table would paint over it, so give that pixel back.
        body: view === 'readable' ? { padding: '1px 0 0' } : undefined
      }}
    >
      {view === 'readable' ? (
        <Table<Claim>
          rowKey='key'
          dataSource={rows}
          columns={columns}
          showHeader={false}
          pagination={false}
          size='middle'
          tableLayout='fixed'
          // The card's own border closes the table; a rule on the last row
          // would double it.
          className='[&_.ant-table-tbody>tr:last-child>td]:!border-b-0'
        />
      ) : (
        <div className='relative'>
          <Typography.Paragraph className='!mb-0'>
            <pre className='!m-0 max-h-[480px] overflow-auto !p-4 !pr-12'>{json}</pre>
          </Typography.Paragraph>
          <Typography.Text copyable={copyable(json)} className='!absolute right-3 top-3' />
        </div>
      )}
    </Card>
  );
};
