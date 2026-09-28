import { useState } from 'react';
import { decimalsHint, validateDecimalInput } from '../lib/decimalInput';
import { depositBalance, type MarketSymbol } from '../lib/market';
import Dialog from './Dialog';

type DepositDialogProps = {
  symbols: MarketSymbol[];
  onClose: () => void;
  onDeposited: () => void;
};

export default function DepositDialog({ symbols, onClose, onDeposited }: DepositDialogProps) {
  const [symbolId, setSymbolId] = useState(symbols[0]?.id);
  const [amount, setAmount] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');

  const symbol = symbols.find((candidate) => candidate.id === symbolId) ?? symbols[0];
  const amountError =
    amount === '' || !symbol ? null : validateDecimalInput(amount, symbol.precision, symbol.code);
  const canSubmit = Boolean(symbol) && amount !== '' && !amountError && !isSubmitting;

  const handleSubmit = async () => {
    if (!canSubmit || !symbol) {
      return;
    }

    setIsSubmitting(true);
    setSubmitError('');

    try {
      await depositBalance(symbol.id, amount);
      onDeposited();
      onClose();
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Unable to complete deposit.');
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog
      title="Deposit"
      titleId="deposit-title"
      className="deposit-dialog"
      isBusy={isSubmitting}
      onClose={onClose}
      onSubmit={() => void handleSubmit()}
    >
      <div className="field-group">
        <label htmlFor="deposit-symbol">Currency</label>
        <select
          id="deposit-symbol"
          value={symbol?.id}
          onChange={(event) => setSymbolId(Number(event.target.value))}
          disabled={isSubmitting}
        >
          {symbols.map((option) => (
            <option key={option.id} value={option.id}>
              {option.code}
            </option>
          ))}
        </select>
      </div>

      <div className="field-group">
        <label htmlFor="deposit-amount">Amount</label>
        <input
          id="deposit-amount"
          name="amount"
          type="text"
          inputMode="decimal"
          autoComplete="off"
          autoFocus
          placeholder="0"
          value={amount}
          onChange={(event) => setAmount(event.target.value)}
          disabled={isSubmitting}
          aria-invalid={amountError ? true : undefined}
          aria-describedby="deposit-amount-message"
        />
        <span id="deposit-amount-message" className={`field-message ${amountError ? 'error' : ''}`}>
          {amountError ?? (symbol ? decimalsHint(symbol.precision) : '')}
        </span>
      </div>

      {submitError ? (
        <p className="form-note error" role="alert">
          {submitError}
        </p>
      ) : null}

      <div className="dialog-actions">
        <button
          type="button"
          className="secondary-action"
          onClick={onClose}
          disabled={isSubmitting}
        >
          Cancel
        </button>
        <button type="submit" className="primary-action" disabled={!canSubmit}>
          {isSubmitting ? 'Depositing...' : 'Deposit'}
        </button>
      </div>
    </Dialog>
  );
}
