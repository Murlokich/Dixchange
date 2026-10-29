import { useState } from 'react';
import { decimalsHint, validateDecimalInput } from '../lib/decimalInput';
import { formatPairLabel, type MarketPair } from '../lib/market';
import { createOrder, type OrderSide } from '../lib/orders';
import Dialog from './Dialog';

type CreateOrderDialogProps = {
  pairs: MarketPair[];
  initialPairId: number | null;
  onClose: () => void;
  onCreated: () => void;
};

const sides: OrderSide[] = ['buy', 'sell'];

export default function CreateOrderDialog({
  pairs,
  initialPairId,
  onClose,
  onCreated,
}: CreateOrderDialogProps) {
  const [pairId, setPairId] = useState(initialPairId ?? pairs[0]?.id);
  const [side, setSide] = useState<OrderSide>('buy');
  const [quantity, setQuantity] = useState('');
  const [price, setPrice] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState('');

  const pair = pairs.find((candidate) => candidate.id === pairId) ?? pairs[0];
  const [base, quote] = pair?.symbols ?? [];

  // Quantity is limited by the base symbol precision, price by the pair's price precision.
  const quantityError =
    quantity === '' || !base ? null : validateDecimalInput(quantity, base.precision, base.code);
  const priceError =
    price === '' || !pair ? null : validateDecimalInput(price, pair.price_precision, 'Price');
  const isComplete = quantity !== '' && price !== '' && !quantityError && !priceError;
  const canSubmit = Boolean(pair && base && quote) && isComplete && !isSubmitting;

  const handleSubmit = async () => {
    if (!canSubmit || !pair) {
      return;
    }

    setIsSubmitting(true);
    setSubmitError('');

    try {
      await createOrder({
        market_pair_id: pair.id,
        side,
        quantity_ordered: quantity,
        price_per_unit: price,
      });
      onCreated();
      onClose();
    } catch (error) {
      setSubmitError(error instanceof Error ? error.message : 'Unable to place order.');
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog
      title="Create order"
      titleId="create-order-dialog-title"
      className="create-order-dialog"
      isBusy={isSubmitting}
      onClose={onClose}
      onSubmit={() => void handleSubmit()}
    >
      <div className="field-group">
        <label htmlFor="order-pair">Pair</label>
        <select
          id="order-pair"
          value={pair?.id}
          onChange={(event) => setPairId(Number(event.target.value))}
          disabled={isSubmitting}
        >
          {pairs.map((option) => (
            <option key={option.id} value={option.id}>
              {formatPairLabel(option)}
            </option>
          ))}
        </select>
      </div>

      <fieldset className="side-toggle" disabled={isSubmitting}>
        <legend>Side</legend>
        {sides.map((option) => (
          <label key={option} className={`side-option ${option}`}>
            <input
              type="radio"
              name="side"
              value={option}
              checked={side === option}
              onChange={() => setSide(option)}
            />
            <span>
              {option === 'buy' ? 'Buy' : 'Sell'} {base?.code}
            </span>
          </label>
        ))}
      </fieldset>

      <div className="field-group">
        <label htmlFor="order-quantity">Quantity ({base?.code})</label>
        <input
          id="order-quantity"
          name="quantity"
          type="text"
          inputMode="decimal"
          autoComplete="off"
          autoFocus
          placeholder="0"
          value={quantity}
          onChange={(event) => setQuantity(event.target.value)}
          disabled={isSubmitting}
          aria-invalid={quantityError ? true : undefined}
          aria-describedby="order-quantity-message"
        />
        <span
          id="order-quantity-message"
          className={`field-message ${quantityError ? 'error' : ''}`}
        >
          {quantityError ??
            (pair && base
              ? `${decimalsHint(base.precision)} · lot size ${pair.lot_size} ${base.code}`
              : '')}
        </span>
      </div>

      <div className="field-group">
        <label htmlFor="order-price">
          Price ({quote?.code} per 1 {base?.code})
        </label>
        <input
          id="order-price"
          name="price"
          type="text"
          inputMode="decimal"
          autoComplete="off"
          placeholder="0"
          value={price}
          onChange={(event) => setPrice(event.target.value)}
          disabled={isSubmitting}
          aria-invalid={priceError ? true : undefined}
          aria-describedby="order-price-message"
        />
        <span id="order-price-message" className={`field-message ${priceError ? 'error' : ''}`}>
          {priceError ??
            (pair && quote
              ? `${decimalsHint(pair.price_precision)} · tick ${pair.price_tick} ${quote.code}`
              : '')}
        </span>
      </div>

      {isComplete && base && quote ? (
        <p className="order-summary">
          {side === 'buy' ? 'Buy' : 'Sell'} {quantity} {base.code} at {price} {quote.code} per{' '}
          {base.code}
        </p>
      ) : null}

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
          {isSubmitting ? 'Placing...' : `Place ${side} order`}
        </button>
      </div>
    </Dialog>
  );
}
