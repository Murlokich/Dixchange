import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
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

const mockBackend = (balances: Record<number, unknown>) =>
  vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
    const url = String(input);
    if (url === `${host}/api/market/pairs`) {
      return jsonResponse(pairsResponse);
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
