import { SplitMode } from '@sneat/extension-splitus-contract';

// Pure client-side share reconciliation for the create-split form.
// All arithmetic is done in integer hundredths (cents for amounts, basis
// points for percentages) so "0.10 + 0.20 == 0.30" holds exactly — the same
// fixed-point discipline the backend applies (decimal.Decimal64p2).

export interface IShareReconciliation {
  readonly ok: boolean;
  /** Explanatory, user-facing message when `ok` is false. */
  readonly error?: string;
}

/**
 * Parses a decimal user input ("33.34", 50) into integer hundredths (3334).
 * Returns undefined for blank, non-numeric or negative values.
 */
export function parseDecimal(
  raw: string | number | null | undefined,
): number | undefined {
  if (raw === null || raw === undefined) {
    return undefined;
  }
  const s = String(raw).trim();
  if (!s) {
    return undefined;
  }
  const n = Number(s);
  if (!Number.isFinite(n) || n < 0) {
    return undefined;
  }
  return Math.round(n * 100);
}

/** Formats integer hundredths as a 2-decimal string: 3334 → "33.34". */
export function formatHundredths(v: number): string {
  return (v / 100).toFixed(2);
}

/** Formats basis points as a trimmed percent number: 3334 → "33.34", 5000 → "50". */
export function formatPercentValue(v: number): string {
  return String(v / 100);
}

/**
 * Live reconciliation of custom shares against the expense total.
 *
 * - `equally`: always ok — there are no share inputs; the backend splits.
 * - `exact-amount`: every share must be entered and sum exactly to the total.
 * - `percentage`: every percentage must be entered and sum exactly to 100%.
 *
 * @param totalHundredths the expense total in hundredths (cents), if entered.
 * @param shares raw share inputs, one per participant (payer included).
 */
export function reconcileShares(
  mode: SplitMode,
  totalHundredths: number | undefined,
  shares: readonly (string | number | null | undefined)[],
): IShareReconciliation {
  if (mode === 'equally') {
    return { ok: true };
  }

  const parsed = shares.map(parseDecimal);
  if (parsed.some((v) => v === undefined)) {
    return {
      ok: false,
      error:
        mode === 'percentage'
          ? 'Enter a percentage for every participant.'
          : 'Enter a share for every participant.',
    };
  }
  const entered = (parsed as number[]).reduce((sum, v) => sum + v, 0);

  if (mode === 'percentage') {
    if (entered === 100_00) {
      return { ok: true };
    }
    return {
      ok: false,
      error: `Percentages sum to ${formatPercentValue(entered)}% — they must sum to exactly 100%.`,
    };
  }

  // exact-amount
  if (totalHundredths === undefined) {
    return { ok: false, error: 'Enter the expense total first.' };
  }
  if (entered === totalHundredths) {
    return { ok: true };
  }
  const delta = entered - totalHundredths;
  const direction =
    delta < 0
      ? `${formatHundredths(-delta)} short of`
      : `${formatHundredths(delta)} over`;
  return {
    ok: false,
    error: `Shares sum to ${formatHundredths(entered)} — ${direction} the ${formatHundredths(totalHundredths)} total.`,
  };
}
