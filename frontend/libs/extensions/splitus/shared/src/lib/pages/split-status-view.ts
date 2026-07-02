import { ISplitParticipant } from '@sneat/extension-splitus-contract';

// Pure view-mapping helpers shared by the splits home list and the split
// details page. Kept side-effect-free and unit-tested directly (see
// split-status-view.spec.ts), the same way new-split-form's split-shares.ts
// isolates its reconciliation logic from the component.

/**
 * Maps a split/participant status string to a badge color. This is the ONLY
 * thing derived from `status` client-side — the status text itself is always
 * rendered verbatim, exactly as returned by getSplit/getSplits, never
 * recomputed or renamed here (splitus#ac:settle-up-single-source-of-truth:
 * Splitus reads settled/unsettled state from Debtus and never keeps its own).
 */
export function statusColor(status: string): 'success' | 'warning' {
  return status === 'settled' ? 'success' : 'warning';
}

// Debtus is a separate app — splitus-app's space shell mounts only the
// splitus routes (see splitus-space.routes.ts), not debtus's, to keep the two
// apps decoupled. So an outstanding share's "Settle in Debtus" affordance
// links out to the debtus.app settle-up surface by an absolute URL rather
// than an in-app route (mounting debtus's routes/providers into splitus-app
// just for this one link would undo that intentional decoupling).
const DEBTUS_APP_BASE_URL = 'https://debtus.app';

/**
 * The settle-up link for one participant row, or undefined when settling
 * doesn't apply: the payer never settles with themselves, a settled share has
 * nothing left to settle, and a participant with no contactID (e.g. a
 * userID-only participant) has no debtus counterparty to link to.
 */
export function settleInDebtusUrl(
  participant: Pick<ISplitParticipant, 'status' | 'isPayer' | 'contactID'>,
  spaceType: string | undefined,
  spaceID: string | undefined,
): string | undefined {
  if (participant.status !== 'outstanding' || participant.isPayer) {
    return undefined;
  }
  const contactID = participant.contactID;
  if (!contactID || !spaceType || !spaceID) {
    return undefined;
  }
  return `${DEBTUS_APP_BASE_URL}/space/${spaceType}/${spaceID}/settle-up?contactID=${encodeURIComponent(contactID)}`;
}

/** "1 member" / "2 members" — used on the splits list rows. */
export function formatMemberCount(count: number): string {
  return `${count} ${count === 1 ? 'member' : 'members'}`;
}
