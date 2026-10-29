import { fetchTrades } from '../lib/trades';
import { useCursorPages } from '../lib/useCursorPages';
import Dialog from './Dialog';
import TradesTable from './TradesTable';

type TradeHistoryDialogProps = {
  pairLabels: Map<number, string>;
  onClose: () => void;
};

// Full trade history, paged with "Load more". Loaded on demand and not polled.
export default function TradeHistoryDialog({ pairLabels, onClose }: TradeHistoryDialogProps) {
  const { items, hasMore, isLoading, error, loadMore } = useCursorPages(fetchTrades);

  return (
    <Dialog
      title="Trade history"
      titleId="trade-history-dialog-title"
      className="dialog-wide"
      onClose={onClose}
    >
      <div className="dialog-scroll">
        {items.length > 0 ? (
          <TradesTable trades={items} pairLabels={pairLabels} />
        ) : !isLoading && !error ? (
          <p className="empty-state">No trades yet.</p>
        ) : null}

        {error ? (
          <p className="form-note error" role="alert">
            {error}
          </p>
        ) : null}

        {hasMore ? (
          <button
            type="button"
            className="secondary-action load-more"
            onClick={loadMore}
            disabled={isLoading}
          >
            {isLoading ? 'Loading…' : error ? 'Retry' : 'Load more'}
          </button>
        ) : null}
      </div>

      <div className="dialog-actions">
        <button type="button" className="secondary-action" onClick={onClose}>
          Close
        </button>
      </div>
    </Dialog>
  );
}
