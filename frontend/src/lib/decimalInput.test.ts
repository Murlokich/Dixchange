import { describe, expect, it } from 'vitest';
import { validateDecimalInput } from './decimalInput';

describe('validateDecimalInput', () => {
  it.each(['1', '10', '10.5', '10.25', '0.01', '007'])('accepts %s with precision 2', (value) => {
    expect(validateDecimalInput(value, 2, 'USD')).toBeNull();
  });

  it.each(['abc', '1,5', '.5', '5.', '-1', '1e3', ' 1', '1 ', '1.2.3', '+1'])(
    'rejects malformed %j',
    (value) => {
      expect(validateDecimalInput(value, 2, 'USD')).toBe(
        'Enter a positive number, e.g. 10 or 10.5',
      );
    },
  );

  it('rejects more decimals than the symbol precision', () => {
    expect(validateDecimalInput('1.234', 2, 'USD')).toBe('USD allows at most 2 decimal places');
    expect(validateDecimalInput('1.23', 1, 'EUR')).toBe('EUR allows at most 1 decimal place');
    expect(validateDecimalInput('1.5', 0, 'KCN')).toBe('KCN accepts whole numbers only');
    expect(validateDecimalInput('1.0', 0, 'KCN')).toBe('KCN accepts whole numbers only');
  });

  it('rejects zero in any form', () => {
    expect(validateDecimalInput('0', 2, 'USD')).toBe('Must be greater than 0');
    expect(validateDecimalInput('0.00', 2, 'USD')).toBe('Must be greater than 0');
  });
});
