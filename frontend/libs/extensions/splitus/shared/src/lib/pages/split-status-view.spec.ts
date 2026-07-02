import { formatMemberCount, settleInDebtusUrl, statusColor } from './split-status-view';

describe('statusColor (splitus#ac:settle-up-single-source-of-truth)', () => {
  it('colors "settled" green', () => {
    expect(statusColor('settled')).toBe('success');
  });

  it('colors "outstanding" amber', () => {
    expect(statusColor('outstanding')).toBe('warning');
  });

  it('falls back to amber for any other status text — never re-derives it', () => {
    // The badge color is the only thing computed from status; the API's raw
    // status string is always rendered as-is by the template, unmodified.
    expect(statusColor('anything-else')).toBe('warning');
  });
});

describe('settleInDebtusUrl (settle affordance decision)', () => {
  const base = { spaceType: 'family', spaceID: 'space1' };

  it('links to the debtus.app settle-up surface for an outstanding, non-payer participant', () => {
    const url = settleInDebtusUrl(
      { status: 'outstanding', isPayer: false, contactID: 'bea' },
      base.spaceType,
      base.spaceID,
    );
    expect(url).toBe(
      'https://debtus.app/space/family/space1/settle-up?contactID=bea',
    );
  });

  it('returns undefined for a settled share — nothing left to settle', () => {
    expect(
      settleInDebtusUrl(
        { status: 'settled', isPayer: false, contactID: 'bea' },
        base.spaceType,
        base.spaceID,
      ),
    ).toBeUndefined();
  });

  it('returns undefined for the payer row', () => {
    expect(
      settleInDebtusUrl(
        { status: 'outstanding', isPayer: true, contactID: 'alex' },
        base.spaceType,
        base.spaceID,
      ),
    ).toBeUndefined();
  });

  it('returns undefined without a contactID to settle against', () => {
    expect(
      settleInDebtusUrl(
        { status: 'outstanding', isPayer: false },
        base.spaceType,
        base.spaceID,
      ),
    ).toBeUndefined();
  });

  it('returns undefined without a resolved space', () => {
    expect(
      settleInDebtusUrl(
        { status: 'outstanding', isPayer: false, contactID: 'bea' },
        undefined,
        undefined,
      ),
    ).toBeUndefined();
  });

  it('encodes the contactID in the URL', () => {
    const url = settleInDebtusUrl(
      { status: 'outstanding', isPayer: false, contactID: 'bea/1' },
      base.spaceType,
      base.spaceID,
    );
    expect(url).toBe(
      'https://debtus.app/space/family/space1/settle-up?contactID=bea%2F1',
    );
  });
});

describe('formatMemberCount', () => {
  it('singular for 1', () => {
    expect(formatMemberCount(1)).toBe('1 member');
  });

  it('plural for 0 and 2+', () => {
    expect(formatMemberCount(0)).toBe('0 members');
    expect(formatMemberCount(3)).toBe('3 members');
  });
});
