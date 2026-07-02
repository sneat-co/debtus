import {
  formatHundredths,
  parseDecimal,
  reconcileShares,
} from './split-shares';

describe('parseDecimal', () => {
  it('parses decimal strings and numbers to hundredths', () => {
    expect(parseDecimal('33.34')).toBe(3334);
    expect(parseDecimal('100')).toBe(10000);
    expect(parseDecimal(0.1)).toBe(10);
    expect(parseDecimal('0')).toBe(0);
  });

  it('returns undefined for blank or invalid values', () => {
    expect(parseDecimal('')).toBeUndefined();
    expect(parseDecimal('  ')).toBeUndefined();
    expect(parseDecimal(null)).toBeUndefined();
    expect(parseDecimal(undefined)).toBeUndefined();
    expect(parseDecimal('abc')).toBeUndefined();
    expect(parseDecimal('-5')).toBeUndefined();
  });
});

describe('formatHundredths', () => {
  it('formats hundredths as a 2-decimal string', () => {
    expect(formatHundredths(3334)).toBe('33.34');
    expect(formatHundredths(10000)).toBe('100.00');
    expect(formatHundredths(0)).toBe('0.00');
  });
});

describe('reconcileShares', () => {
  describe('equally', () => {
    it('always reconciles — the backend computes equal shares', () => {
      expect(reconcileShares('equally', 10000, [])).toEqual({ ok: true });
      expect(reconcileShares('equally', undefined, ['1'])).toEqual({
        ok: true,
      });
    });
  });

  describe('exact-amount', () => {
    it('reconciles when shares sum exactly to the total', () => {
      const r = reconcileShares('exact-amount', 10000, ['50.00', '30', '20']);
      expect(r.ok).toBe(true);
      expect(r.error).toBeUndefined();
    });

    it('rejects with a "short" message when shares sum below the total', () => {
      const r = reconcileShares('exact-amount', 10000, ['50.00', '30.00']);
      expect(r.ok).toBe(false);
      expect(r.error).toContain('80.00');
      expect(r.error).toContain('20.00');
      expect(r.error).toContain('short');
      expect(r.error).toContain('100.00');
    });

    it('rejects with an "over" message when shares sum above the total', () => {
      const r = reconcileShares('exact-amount', 10000, ['60', '30', '20']);
      expect(r.ok).toBe(false);
      expect(r.error).toContain('110.00');
      expect(r.error).toContain('10.00');
      expect(r.error).toContain('over');
    });

    it('rejects when a share is blank or invalid', () => {
      expect(reconcileShares('exact-amount', 10000, ['50', '']).ok).toBe(
        false,
      );
      expect(
        reconcileShares('exact-amount', 10000, ['50', '']).error,
      ).toContain('every participant');
      expect(reconcileShares('exact-amount', 10000, ['50', 'x']).ok).toBe(
        false,
      );
    });

    it('rejects when the total is not set yet', () => {
      const r = reconcileShares('exact-amount', undefined, ['50', '50']);
      expect(r.ok).toBe(false);
      expect(r.error).toContain('total');
    });

    it('is exact to the cent — no float drift', () => {
      // 0.1 + 0.2 !== 0.3 in floats; in hundredths it must reconcile.
      const r = reconcileShares('exact-amount', 30, ['0.1', '0.2']);
      expect(r.ok).toBe(true);
    });
  });

  describe('percentage', () => {
    it('reconciles when percentages sum exactly to 100%', () => {
      const r = reconcileShares('percentage', 10000, [
        '33.33',
        '33.33',
        '33.34',
      ]);
      expect(r.ok).toBe(true);
    });

    it('rejects when percentages do not sum to 100%', () => {
      const r = reconcileShares('percentage', 10000, ['45', '45']);
      expect(r.ok).toBe(false);
      expect(r.error).toContain('90');
      expect(r.error).toContain('100%');
    });

    it('does not require the total amount to validate percentages', () => {
      expect(reconcileShares('percentage', undefined, ['50', '50']).ok).toBe(
        true,
      );
    });

    it('rejects blank or invalid percentages', () => {
      const r = reconcileShares('percentage', 10000, ['50', '']);
      expect(r.ok).toBe(false);
      expect(r.error).toContain('every participant');
    });
  });
});
