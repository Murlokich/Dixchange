// Format checks on the raw text only. The value is never parsed into a number, so the
// string the user typed is exactly what gets sent to the backend.
const decimalPattern = /^\d+(\.\d+)?$/;

export function validateDecimalInput(
  value: string,
  precision: number,
  code: string,
): string | null {
  if (!decimalPattern.test(value)) {
    return 'Enter a positive number, e.g. 10 or 10.5';
  }

  const decimals = value.split('.')[1] ?? '';
  if (decimals.length > precision) {
    return precision === 0
      ? `${code} accepts whole numbers only`
      : `${code} allows at most ${precision} decimal place${precision === 1 ? '' : 's'}`;
  }

  if (!/[1-9]/.test(value)) {
    return 'Must be greater than 0';
  }

  return null;
}

export function decimalsHint(precision: number) {
  if (precision === 0) {
    return 'Whole numbers only';
  }
  return `Up to ${precision} decimal place${precision === 1 ? '' : 's'}`;
}
