import { isPromotionRequestPhaseTerminal } from '@ui/features/stage/utils/promotion-request';
import {
  FreightCollection,
  Health,
  PromotionRequest,
  PromotionRequestSummary,
  Stage,
  Target
} from '@ui/gen/api/v2/models';

// Severity ranks a row for ordering: rounds that went wrong come first,
// rounds still moving second, settled rows last.
export const SEVERITY_FAILED = 0;
export const SEVERITY_ACTIVE = 1;
export const SEVERITY_SETTLED = 2;

export type FleetRow = {
  target: Target;
  // The round whose outcome the row reports, if the Stage has had one.
  request?: PromotionRequest;
  // Whether that round named this Target. A Target that joined the Stage
  // after the round was created is governed but not included.
  included: boolean;
  // The child Promotion promoting to this Target in that round, once created.
  promotion?: string;
  // The phase to show. Comes from the child Promotion when there is one; from
  // the round itself when it ended before a child could be created; and is
  // Pending while the round has yet to reach this Target.
  phase?: string;
  // What the Target is running for this Stage, from the Target's own status.
  // Absent until a Promotion from the Stage to this Target has succeeded.
  currentFreight?: FreightCollection;
  // Whether currentFreight matches the Stage's own current collection. False
  // when the Target has fallen behind; undefined when either side is unknown.
  upToDate?: boolean;
  // The Target's health with respect to this Stage's Freight.
  health?: Health;
};

// rowSeverity ranks a row by the phase it shows.
export const rowSeverity = (row: FleetRow): number => {
  switch (row.phase) {
    case 'Failed':
    case 'Errored':
      return SEVERITY_FAILED;
    case 'Pending':
    case 'Running':
      return SEVERITY_ACTIVE;
    default:
      return SEVERITY_SETTLED;
  }
};

// promotionPhaseFor resolves what a Target's Promotion cell shows for a round.
const promotionPhaseFor = (
  target: Target,
  request?: PromotionRequest
): Pick<FleetRow, 'included' | 'promotion' | 'phase'> => {
  const name = target.metadata?.name;
  const included = !!request?.spec?.targets?.some((named) => named.name === name);
  if (!request || !included) {
    return { included };
  }
  const status = request.status?.targets?.find((named) => named.name === name);
  if (status) {
    // A child that exists but has not been picked up yet has no phase; it is
    // Pending in every sense that matters here.
    return { included, promotion: status.promotion, phase: status.phase || 'Pending' };
  }
  const requestPhase = request.status?.phase;
  return {
    included,
    phase: isPromotionRequestPhaseTerminal(requestPhase) ? requestPhase : 'Pending'
  };
};

// fleetRows builds one row per Target the Stage governs, reading the round's
// outcome from the PromotionRequest and the Target's standing state from its
// own status. Rows with trouble sort first, then by Target name.
export const fleetRows = (
  stage: Stage,
  targets: Target[],
  request?: PromotionRequest
): FleetRow[] => {
  const stageName = stage.metadata?.name || '';
  const stageCollectionID = stage.status?.freightHistory?.[0]?.id;
  return targets
    .map<FleetRow>((target) => {
      const forStage = target.status?.stages?.[stageName];
      const currentFreight = forStage?.currentFreight;
      return {
        target,
        request,
        ...promotionPhaseFor(target, request),
        currentFreight,
        upToDate:
          currentFreight?.id && stageCollectionID
            ? currentFreight.id === stageCollectionID
            : undefined,
        health: forStage?.health
      };
    })
    .sort((lhs, rhs) => {
      const bySeverity = rowSeverity(lhs) - rowSeverity(rhs);
      if (bySeverity !== 0) {
        return bySeverity;
      }
      return (lhs.target.metadata?.name || '').localeCompare(rhs.target.metadata?.name || '');
    });
};

// freightNames lists the Freight a collection holds, one per origin, in a
// stable order.
export const freightNames = (collection?: FreightCollection): string[] =>
  Object.values(collection?.items || {})
    .map((reference) => reference.name || '')
    .filter(Boolean)
    .sort();

// rowsSummary tallies the rows' Promotion phases in the shape of a
// PromotionRequest summary, so the round's progress bar is drawn from the same
// per-Target phases the table shows and the two cannot disagree. Rows the round
// did not name have no phase and are not counted.
export const rowsSummary = (rows: FleetRow[]): PromotionRequestSummary => {
  const summary: PromotionRequestSummary = {};
  for (const row of rows) {
    switch (row.phase) {
      case 'Pending':
        summary.pending = (summary.pending || 0) + 1;
        break;
      case 'Running':
        summary.running = (summary.running || 0) + 1;
        break;
      case 'Succeeded':
        summary.succeeded = (summary.succeeded || 0) + 1;
        break;
      case 'Failed':
        summary.failed = (summary.failed || 0) + 1;
        break;
      case 'Errored':
        summary.errored = (summary.errored || 0) + 1;
        break;
      case 'Aborted':
        summary.aborted = (summary.aborted || 0) + 1;
        break;
      default:
        break;
    }
  }
  return summary;
};

// filterRows narrows rows by a free-text needle matched against the Target's
// name and its labels as key=value, and by a Promotion phase. Either may be
// empty.
export const filterRows = (rows: FleetRow[], needle: string, phase?: string): FleetRow[] => {
  const text = needle.trim().toLowerCase();
  return rows.filter((row) => {
    if (phase && (row.phase || 'Pending') !== phase) {
      return false;
    }
    if (!text) {
      return true;
    }
    const haystack = [
      row.target.metadata?.name || '',
      ...Object.entries(row.target.metadata?.labels || {}).map(([k, v]) => `${k}=${v}`)
    ]
      .join(' ')
      .toLowerCase();
    return haystack.includes(text);
  });
};

// partitionRows splits rows into those that need attention -- anything that
// did not succeed, plus anything still moving -- and those that succeeded.
// When a round has trouble, the succeeded rows can collapse into one line so
// the failures are the table.
export const partitionRows = (
  rows: FleetRow[]
): { attention: FleetRow[]; succeeded: FleetRow[] } => {
  const attention: FleetRow[] = [];
  const succeeded: FleetRow[] = [];
  for (const row of rows) {
    (row.phase === 'Succeeded' ? succeeded : attention).push(row);
  }
  return { attention, succeeded };
};

// namesPreview lists the first few Target names of a set and how many more
// there are, for a collapsed group's one-line summary.
export const namesPreview = (rows: FleetRow[], shown = 3): string => {
  const names = rows.map((row) => row.target.metadata?.name || '');
  if (names.length <= shown) {
    return names.join(', ');
  }
  return `${names.slice(0, shown).join(', ')} and ${names.length - shown} more`;
};
