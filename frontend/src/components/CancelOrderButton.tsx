import { useEffect, useState } from 'react';
import { cancelOrder, type Order } from '../lib/orders';

type CancelOrderButtonProps = {
  order: Order;
  onCancelled: () => void;
};

// How long the "confirm" state waits for a second click before reverting.
const confirmTimeoutMs = 3000;

type CancelState = 'idle' | 'confirming' | 'cancelling';

// Two-step cancel: the first click arms the button, the second one sends the request.
export default function CancelOrderButton({ order, onCancelled }: CancelOrderButtonProps) {
  const [state, setState] = useState<CancelState>('idle');
  const [error, setError] = useState('');

  useEffect(() => {
    if (state !== 'confirming') {
      return;
    }
    const timer = window.setTimeout(() => setState('idle'), confirmTimeoutMs);
    return () => window.clearTimeout(timer);
  }, [state]);

  const handleClick = async () => {
    if (state === 'idle') {
      setError('');
      setState('confirming');
      return;
    }
    if (state !== 'confirming') {
      return;
    }

    setState('cancelling');
    try {
      await cancelOrder(order.id);
      onCancelled();
    } catch (cancelError) {
      setError(cancelError instanceof Error ? cancelError.message : 'Unable to cancel order.');
      // The order may have filled or been cancelled meanwhile, so show the current state.
      onCancelled();
    } finally {
      setState('idle');
    }
  };

  // A partially filled order only cancels what is left, so the confirmation says how much.
  const confirmLabel =
    order.status === 'partially_filled' ? `Cancel ${order.remaining_quantity} left` : 'Confirm';
  const label =
    state === 'cancelling' ? 'Cancelling…' : state === 'confirming' ? confirmLabel : 'Cancel';

  return (
    <div className="cancel-order">
      <button
        type="button"
        className={`cancel-button ${state === 'confirming' ? 'confirming' : ''}`}
        onClick={() => void handleClick()}
        disabled={state === 'cancelling'}
      >
        {label}
      </button>
      {error ? (
        <span className="cancel-error" role="alert">
          {error}
        </span>
      ) : null}
    </div>
  );
}
