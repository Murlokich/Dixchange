import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import DashboardPage from './DashboardPage';

const host = import.meta.env.VITE_BACKEND_HOST;

const jsonResponse = (body: unknown) =>
  ({ ok: true, status: 200, json: async () => body }) as Response;

const pairsResponse = {
  pairs: [
    {
      id: 1,
      symbols: [
        { id: 1, code: 'USD', precision: 2 },
        { id: 2, code: 'KCN', precision: 0 },
      ],
      lot_size: '1.00',
      price_tick: '1',
      price_precision: 0,
    },
    {
      id: 2,
      symbols: [
        { id: 3, code: 'EUR', precision: 2 },
        { id: 2, code: 'KCN', precision: 0 },
      ],
      lot_size: '0.01',
      price_tick: '0.5',
      price_precision: 1,
    },
  ],
};

const balanceResponses: Record<number, unknown> = {
  1: {
    pair_id: 1,
    base: { main: '8.00', reserved: '2.00' },
    quote: { main: '500', reserved: '150' },
  },
  2: {
    pair_id: 2,
    base: { main: '40.50', reserved: '0.00' },
    quote: { main: '75', reserved: '5' },
  },
};

const ordersUrl = `${host}/api/orders?status=open,partially_filled`;

const ordersResponse = {
  orders: [
    {
      id: '01a0cdca-9934-71a2-9983-73541b0e6921',
      pair_id: 1,
      side: 'buy',
      quantity_ordered: '3.00',
      quantity_filled: '0.00',
      remaining_quantity: '3.00',
      price_per_unit: '7',
      status: 'open',
      created_at: '2026-09-23T10:23:21Z',
      last_updated_at: null,
    },
    {
      id: '01a0cdbe-6d4c-7106-bda9-5f35959e6b96',
      pair_id: 2,
      side: 'sell',
      quantity_ordered: '6.00',
      quantity_filled: '1.50',
      remaining_quantity: '4.50',
      price_per_unit: '4.5',
      status: 'partially_filled',
      created_at: '2026-09-23T10:10:04Z',
      last_updated_at: '2026-09-24T08:00:00Z',
    },
    {
      id: '01a0cdbe-0000-7106-bda9-5f35959e6b96',
      pair_id: 9,
      side: 'buy',
      quantity_ordered: '9.00',
      quantity_filled: '0.00',
      remaining_quantity: '9.00',
      price_per_unit: '3',
      status: 'open',
      created_at: '2026-09-22T09:00:00Z',
      last_updated_at: null,
    },
  ],
};

const tradesUrl = `${host}/api/trades`;

const makeTrade = (id: string, amount: string) => ({
  id,
  pair_id: 1,
  order_id: `order-${id}`,
  side: 'buy',
  amount,
  price: '3',
  total: '36',
  executed_at: '2026-09-28T19:39:56Z',
});

const tradesResponse = {
  trades: [makeTrade('t1', '12.00'), { ...makeTrade('t2', '13.00'), pair_id: 2, side: 'sell' }],
  next_cursor: 'cursor-1',
};

const mockBackend = (balances: Record<number, unknown>, orders: unknown = ordersResponse) =>
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input);
    if (url === `${host}/api/market/pairs`) {
      return jsonResponse(pairsResponse);
    }
    if (url === ordersUrl) {
      return jsonResponse(orders);
    }
    if (url === tradesUrl) {
      return jsonResponse(tradesResponse);
    }
    return jsonResponse(balances[Number(url.split('/').pop())]);
  });

describe('DashboardPage pair selector', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads pairs once, shows balances for the selected pair and switches pairs', async () => {
    const fetchSpy = mockBackend(balanceResponses);

    render(<DashboardPage />);

    const trigger = await screen.findByRole('button', { name: /USD\/KCN/ });
    expect(trigger.textContent).toContain('#1');
    expect(await screen.findByText('8.00')).toBeTruthy();
    expect(screen.getByText('2.00 reserved')).toBeTruthy();
    expect(screen.getByText('500')).toBeTruthy();
    expect(screen.getByText('150 reserved')).toBeTruthy();
    expect(fetchSpy).toHaveBeenCalledWith(`${host}/api/balance/1`, expect.anything());

    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole('menuitemradio', { name: /EUR\/KCN/ }));

    expect(await screen.findByText('40.50')).toBeTruthy();
    expect(screen.getByText('EUR')).toBeTruthy();
    expect(screen.getByText('5 reserved')).toBeTruthy();
    expect(screen.queryByText('8.00')).toBeNull();
    expect(
      fetchSpy.mock.calls.filter(([url]) => String(url).endsWith('/api/market/pairs')),
    ).toHaveLength(1);
  });

  it('ignores a balance response for a different pair', async () => {
    const fetchSpy = mockBackend({ 1: balanceResponses[2] });

    render(<DashboardPage />);

    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(`${host}/api/balance/1`, expect.anything()),
    );
    await screen.findByRole('button', { name: /USD\/KCN/ });
    expect(screen.queryByText('40.50')).toBeNull();
    expect(screen.getAllByText('— reserved')).toHaveLength(2);
  });

  it('polls the balance at the configured interval and shows fresh values', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let call = 0;
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      if (String(input) === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (String(input) === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (String(input) === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      call += 1;
      return jsonResponse({
        pair_id: 1,
        base: { main: `${call}.00`, reserved: '0.00' },
        quote: { main: '500', reserved: '0' },
      });
    });

    try {
      render(<DashboardPage />);
      expect(await screen.findByText('1.00')).toBeTruthy();

      await vi.advanceTimersByTimeAsync(5000);
      expect(await screen.findByText('2.00')).toBeTruthy();
      expect(
        fetchSpy.mock.calls.filter(([url]) => String(url) === `${host}/api/balance/1`),
      ).toHaveLength(2);
    } finally {
      vi.useRealTimers();
    }
  });
});

describe('DashboardPage deposit', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('validates the amount per symbol precision and posts the raw string', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (url === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (url === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      if (url === `${host}/api/balance/deposit`) {
        return jsonResponse({});
      }
      return jsonResponse(balanceResponses[1]);
    });

    render(<DashboardPage />);
    await screen.findByText('8.00');

    fireEvent.click(screen.getByRole('button', { name: /deposit/i }));
    const dialog = screen.getByRole('dialog', { name: /deposit/i });
    const amount = within(dialog).getByLabelText(/amount/i);
    const submit = within(dialog).getByRole('button', { name: /^deposit$/i });

    expect(within(dialog).getByText('Up to 2 decimal places')).toBeTruthy();
    expect(submit).toHaveProperty('disabled', true);

    fireEvent.change(amount, { target: { value: '1.234' } });
    expect(amount.getAttribute('aria-invalid')).toBe('true');
    expect(within(dialog).getByText('USD allows at most 2 decimal places')).toBeTruthy();
    expect(submit).toHaveProperty('disabled', true);

    fireEvent.change(within(dialog).getByLabelText(/currency/i), { target: { value: '2' } });
    fireEvent.change(amount, { target: { value: '1.5' } });
    expect(within(dialog).getByText('KCN accepts whole numbers only')).toBeTruthy();

    fireEvent.change(amount, { target: { value: '25' } });
    expect(amount.getAttribute('aria-invalid')).toBeNull();
    expect(submit).toHaveProperty('disabled', false);

    const balanceCallsBefore = fetchSpy.mock.calls.filter(([url]) =>
      String(url).endsWith('/api/balance/1'),
    ).length;
    fireEvent.click(submit);

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(fetchSpy).toHaveBeenCalledWith(`${host}/api/balance/deposit`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ amount: '25', symbol_id: 2 }),
    });
    await waitFor(() =>
      expect(
        fetchSpy.mock.calls.filter(([url]) => String(url).endsWith('/api/balance/1')).length,
      ).toBeGreaterThan(balanceCallsBefore),
    );
  });

  it('keeps the dialog open and shows the error when the deposit fails', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (url === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (url === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      if (url === `${host}/api/balance/deposit`) {
        return { ok: false, status: 400 } as Response;
      }
      return jsonResponse(balanceResponses[1]);
    });

    render(<DashboardPage />);
    await screen.findByText('8.00');

    fireEvent.click(screen.getByRole('button', { name: /deposit/i }));
    const dialog = screen.getByRole('dialog', { name: /deposit/i });
    fireEvent.change(within(dialog).getByLabelText(/amount/i), { target: { value: '10.50' } });
    fireEvent.click(within(dialog).getByRole('button', { name: /^deposit$/i }));

    expect(await within(dialog).findByRole('alert')).toHaveProperty(
      'textContent',
      'Deposit request failed with status 400',
    );
    expect(screen.getByRole('dialog')).toBeTruthy();
  });
});

describe('DashboardPage open orders', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('renders open orders across all pairs from the backend', async () => {
    const fetchSpy = mockBackend(balanceResponses);

    render(<DashboardPage />);

    const section = screen.getByRole('region', { name: /open orders/i });
    const table = await within(section).findByRole('table');
    await within(table).findByText('USD/KCN');
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows).toHaveLength(3);

    expect(rows[0].textContent).toContain('buy');
    expect(rows[0].textContent).toContain('7');
    expect(rows[0].textContent).toContain('3.00 / 3.00');
    expect(within(rows[0]).getByTitle('Not updated since creation')).toBeTruthy();

    expect(within(rows[1]).getByText('EUR/KCN')).toBeTruthy();
    expect(within(rows[1]).getByText('sell').className).toBe('side-sell');
    expect(rows[1].textContent).toContain('4.50 / 6.00');
    expect(within(rows[1]).getByTitle(/^Last updated /)).toBeTruthy();

    expect(within(rows[2]).getByText('Pair #9')).toBeTruthy();

    expect(
      screen.getByText('Open orders', { selector: '.stat-label' }).nextSibling?.textContent,
    ).toBe('3');
    expect(fetchSpy).toHaveBeenCalledWith(ordersUrl, expect.anything());
  });

  it('shows the empty state when there are no open orders', async () => {
    mockBackend(balanceResponses, { orders: [] });

    render(<DashboardPage />);

    const section = screen.getByRole('region', { name: /open orders/i });
    expect(await within(section).findByText('No open orders yet.')).toBeTruthy();
    expect(within(section).queryByRole('table')).toBeNull();
  });
});

describe('DashboardPage create order', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const mockOrderBackend = (createResponse: Response) =>
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (url === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (url === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      if (url === `${host}/api/orders`) {
        return createResponse;
      }
      return jsonResponse(balanceResponses[Number(url.split('/').pop())]);
    });

  const openDialog = async () => {
    await screen.findByText('8.00');
    fireEvent.click(screen.getByRole('button', { name: /create order/i }));
    return screen.getByRole('dialog', { name: /create order/i });
  };

  it('labels quantity and price against the base, validates them and posts the order', async () => {
    const fetchSpy = mockOrderBackend(jsonResponse({}));

    render(<DashboardPage />);
    const dialog = await openDialog();

    const quantity = within(dialog).getByLabelText('Quantity (USD)');
    const price = within(dialog).getByLabelText('Price (KCN per 1 USD)');
    const submit = within(dialog).getByRole('button', { name: /place buy order/i });
    expect(within(dialog).getByText('Up to 2 decimal places · lot size 1.00 USD')).toBeTruthy();
    expect(within(dialog).getByText('Whole numbers only · tick 1 KCN')).toBeTruthy();
    expect(submit).toHaveProperty('disabled', true);

    fireEvent.change(quantity, { target: { value: '1.234' } });
    fireEvent.change(price, { target: { value: '100.5' } });
    expect(quantity.getAttribute('aria-invalid')).toBe('true');
    expect(within(dialog).getByText('USD allows at most 2 decimal places')).toBeTruthy();
    expect(within(dialog).getByText('Price accepts whole numbers only')).toBeTruthy();
    expect(submit).toHaveProperty('disabled', true);

    fireEvent.change(quantity, { target: { value: '1.5' } });
    fireEvent.change(price, { target: { value: '100' } });
    fireEvent.click(within(dialog).getByLabelText('Sell USD'));
    expect(within(dialog).getByText('Sell 1.5 USD at 100 KCN per USD')).toBeTruthy();

    const ordersCallsBefore = fetchSpy.mock.calls.filter(([url]) => url === ordersUrl).length;
    fireEvent.click(within(dialog).getByRole('button', { name: /place sell order/i }));

    await waitFor(() => expect(screen.queryByRole('dialog')).toBeNull());
    expect(fetchSpy).toHaveBeenCalledWith(`${host}/api/orders`, {
      method: 'POST',
      credentials: 'include',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        market_pair_id: 1,
        side: 'sell',
        quantity_ordered: '1.5',
        price_per_unit: '100',
      }),
    });
    await waitFor(() =>
      expect(fetchSpy.mock.calls.filter(([url]) => url === ordersUrl).length).toBeGreaterThan(
        ordersCallsBefore,
      ),
    );
  });

  it('uses the chosen pair for labels, precision and the request', async () => {
    const fetchSpy = mockOrderBackend(jsonResponse({}));

    render(<DashboardPage />);
    const dialog = await openDialog();

    fireEvent.change(within(dialog).getByLabelText('Pair'), { target: { value: '2' } });
    fireEvent.change(within(dialog).getByLabelText('Quantity (EUR)'), {
      target: { value: '2.25' },
    });
    fireEvent.change(within(dialog).getByLabelText('Price (KCN per 1 EUR)'), {
      target: { value: '4.5' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: /place buy order/i }));

    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(
        `${host}/api/orders`,
        expect.objectContaining({
          body: JSON.stringify({
            market_pair_id: 2,
            side: 'buy',
            quantity_ordered: '2.25',
            price_per_unit: '4.5',
          }),
        }),
      ),
    );
  });

  it('keeps the dialog open and shows the error when the order is rejected', async () => {
    mockOrderBackend({ ok: false, status: 422 } as Response);

    render(<DashboardPage />);
    const dialog = await openDialog();

    fireEvent.change(within(dialog).getByLabelText('Quantity (USD)'), { target: { value: '1' } });
    fireEvent.change(within(dialog).getByLabelText('Price (KCN per 1 USD)'), {
      target: { value: '100' },
    });
    fireEvent.click(within(dialog).getByRole('button', { name: /place buy order/i }));

    expect(await within(dialog).findByRole('alert')).toHaveProperty(
      'textContent',
      'Order request failed with status 422',
    );
    expect(screen.getByRole('dialog')).toBeTruthy();
  });
});

describe('DashboardPage trade history', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows the latest trades in the dashboard widget', async () => {
    const fetchSpy = mockBackend(balanceResponses);

    render(<DashboardPage />);

    const section = screen.getByRole('region', { name: /trade history/i });
    const table = await within(section).findByRole('table');
    const rows = within(table).getAllByRole('row').slice(1);
    expect(rows).toHaveLength(2);
    expect(within(rows[0]).getByText('USD/KCN')).toBeTruthy();
    expect(rows[0].textContent).toContain('12.00');
    expect(within(rows[1]).getByText('EUR/KCN')).toBeTruthy();
    expect(within(rows[1]).getByText('sell').className).toBe('side-sell');
    expect(fetchSpy).toHaveBeenCalledWith(tradesUrl, expect.anything());
  });

  it('pages through the full history with load more and drops duplicates', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (url === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (url === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      if (url === `${tradesUrl}?cursor=cursor-1`) {
        // Overlaps the first page by one trade, which must not be shown twice.
        return jsonResponse({
          trades: [makeTrade('t2', '13.00'), makeTrade('t3', '14.00')],
          next_cursor: 'cursor-2',
        });
      }
      if (url === `${tradesUrl}?cursor=cursor-2`) {
        return jsonResponse({ trades: [makeTrade('t4', '15.00')], next_cursor: null });
      }
      return jsonResponse(balanceResponses[1]);
    });

    render(<DashboardPage />);
    await screen.findByText('8.00');

    fireEvent.click(screen.getByRole('button', { name: /view all/i }));
    const dialog = screen.getByRole('dialog', { name: /trade history/i });

    await within(dialog).findByText('12.00');
    fireEvent.click(within(dialog).getByRole('button', { name: /load more/i }));
    await within(dialog).findByText('14.00');
    expect(within(dialog).getAllByText('13.00')).toHaveLength(1);

    fireEvent.click(within(dialog).getByRole('button', { name: /load more/i }));
    await within(dialog).findByText('15.00');
    expect(within(within(dialog).getByRole('table')).getAllByRole('row')).toHaveLength(5);
    expect(within(dialog).queryByRole('button', { name: /load more/i })).toBeNull();
    expect(fetchSpy).toHaveBeenCalledWith(`${tradesUrl}?cursor=cursor-2`, expect.anything());

    fireEvent.click(within(dialog).getByRole('button', { name: /close/i }));
    expect(screen.queryByRole('dialog')).toBeNull();
  });

  it('keeps loaded trades and offers a retry when a page fails', async () => {
    let failNextPage = true;
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (url === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (url === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      if (url === `${tradesUrl}?cursor=cursor-1`) {
        if (failNextPage) {
          failNextPage = false;
          return { ok: false, status: 500 } as Response;
        }
        return jsonResponse({ trades: [makeTrade('t3', '14.00')], next_cursor: null });
      }
      return jsonResponse(balanceResponses[1]);
    });

    render(<DashboardPage />);
    await screen.findByText('8.00');

    fireEvent.click(screen.getByRole('button', { name: /view all/i }));
    const dialog = screen.getByRole('dialog', { name: /trade history/i });
    await within(dialog).findByText('12.00');

    fireEvent.click(within(dialog).getByRole('button', { name: /load more/i }));
    expect(await within(dialog).findByRole('alert')).toHaveProperty(
      'textContent',
      'Trades request failed with status 500',
    );
    expect(within(dialog).getByText('12.00')).toBeTruthy();

    fireEvent.click(within(dialog).getByRole('button', { name: /retry/i }));
    await within(dialog).findByText('14.00');
    expect(within(dialog).queryByRole('alert')).toBeNull();
  });
});

describe('DashboardPage cancel order', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  const openOrderId = '01a0cdca-9934-71a2-9983-73541b0e6921';
  const cancelUrl = `${host}/api/orders/${openOrderId}/cancel`;

  const mockCancelBackend = (cancelResponse: Response) =>
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse(pairsResponse);
      }
      if (url === ordersUrl) {
        return jsonResponse(ordersResponse);
      }
      if (url === tradesUrl) {
        return jsonResponse(tradesResponse);
      }
      if (url.endsWith('/cancel')) {
        return cancelResponse;
      }
      return jsonResponse(balanceResponses[1]);
    });

  const findOrderRows = async () => {
    const section = screen.getByRole('region', { name: /open orders/i });
    const table = await within(section).findByRole('table');
    return within(table).getAllByRole('row').slice(1);
  };

  it('asks for confirmation, then cancels and refreshes orders and balance', async () => {
    const fetchSpy = mockCancelBackend(jsonResponse({}));

    render(<DashboardPage />);
    const [row] = await findOrderRows();
    await screen.findByText('8.00');

    fireEvent.click(within(row).getByRole('button', { name: 'Cancel' }));
    expect(fetchSpy).not.toHaveBeenCalledWith(cancelUrl, expect.anything());

    const countCalls = (match: (url: string) => boolean) =>
      fetchSpy.mock.calls.filter(([url]) => match(String(url))).length;
    const ordersBefore = countCalls((url) => url === ordersUrl);
    const balanceBefore = countCalls((url) => url.endsWith('/api/balance/1'));

    fireEvent.click(within(row).getByRole('button', { name: 'Confirm' }));

    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(cancelUrl, {
        method: 'POST',
        credentials: 'include',
      }),
    );
    await waitFor(() => {
      expect(countCalls((url) => url === ordersUrl)).toBeGreaterThan(ordersBefore);
      expect(countCalls((url) => url.endsWith('/api/balance/1'))).toBeGreaterThan(balanceBefore);
    });
  });

  it('says how much is left when cancelling a partially filled order', async () => {
    mockCancelBackend(jsonResponse({}));

    render(<DashboardPage />);
    const rows = await findOrderRows();

    fireEvent.click(within(rows[1]).getByRole('button', { name: 'Cancel' }));
    expect(within(rows[1]).getByRole('button', { name: 'Cancel 4.50 left' })).toBeTruthy();
  });

  it('reverts the confirmation after a few seconds without a second click', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const fetchSpy = mockCancelBackend(jsonResponse({}));

    try {
      render(<DashboardPage />);
      const [row] = await findOrderRows();

      fireEvent.click(within(row).getByRole('button', { name: 'Cancel' }));
      expect(within(row).getByRole('button', { name: 'Confirm' })).toBeTruthy();

      await act(() => vi.advanceTimersByTimeAsync(3000));
      expect(within(row).getByRole('button', { name: 'Cancel' })).toBeTruthy();
      expect(fetchSpy).not.toHaveBeenCalledWith(cancelUrl, expect.anything());
    } finally {
      vi.useRealTimers();
    }
  });

  it('shows the error in the row when the backend rejects the cancel', async () => {
    mockCancelBackend({ ok: false, status: 409 } as Response);

    render(<DashboardPage />);
    const [row] = await findOrderRows();

    fireEvent.click(within(row).getByRole('button', { name: 'Cancel' }));
    fireEvent.click(within(row).getByRole('button', { name: 'Confirm' }));

    expect(await within(row).findByRole('alert')).toHaveProperty(
      'textContent',
      'Cancel failed with status 409',
    );
    expect(within(row).getByRole('button', { name: 'Cancel' })).toBeTruthy();
  });
});
