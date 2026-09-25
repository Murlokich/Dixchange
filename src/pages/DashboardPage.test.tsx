import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import DashboardPage from './DashboardPage';

const jsonResponse = (body: unknown) =>
  ({ ok: true, status: 200, json: async () => body }) as Response;

describe('DashboardPage pair selector', () => {
  it('loads pairs once, shows balances for the selected pair and switches pairs', async () => {
    const host = import.meta.env.VITE_BACKEND_HOST;
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input);
      if (url === `${host}/api/market/pairs`) {
        return jsonResponse({
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
        });
      }
      const symbolId = Number(url.split('/').pop());
      return jsonResponse({ amount: symbolId * 1000 + 0.25 });
    });

    render(<DashboardPage />);

    const trigger = await screen.findByRole('button', { name: /USD\/KCN/ });
    expect(trigger.textContent).toContain('#1');
    expect(await screen.findByText('1,000.25')).toBeTruthy();
    expect(await screen.findByText('2,000.25')).toBeTruthy();

    fireEvent.click(trigger);
    fireEvent.click(screen.getByRole('menuitemradio', { name: /EUR\/KCN/ }));

    expect(await screen.findByText('EUR')).toBeTruthy();
    expect(await screen.findByText('3,000.25')).toBeTruthy();
    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(`${host}/api/balances/3`, expect.anything()),
    );
    expect(
      fetchSpy.mock.calls.filter(([url]) => String(url).endsWith('/api/market/pairs')),
    ).toHaveLength(1);

    fetchSpy.mockRestore();
  });
});
