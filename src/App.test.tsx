import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import App from './App';

describe('App', () => {
  it('renders the login page heading', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: /sign in to your account/i })).toBeTruthy();
  });

  it('sends the register request to the configured backend host', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      status: 201,
    } as Response);

    render(<App />);

    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: 'kosta@example.com' },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: 'HelloWorld12344!' },
    });
    fireEvent.click(screen.getByRole('button', { name: /sign up/i }));

    await waitFor(() => expect(fetchSpy).toHaveBeenCalled());

    expect(fetchSpy).toHaveBeenCalledWith(`${import.meta.env.VITE_BACKEND_HOST}/api/auth/register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        email: 'kosta@example.com',
        password: 'HelloWorld12344!',
      }),
    });

    fetchSpy.mockRestore();
  });
});