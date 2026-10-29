import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import App from './App';

const renderApp = () =>
  render(
    <MemoryRouter>
      <App />
    </MemoryRouter>,
  );

describe('App', () => {
  it('renders the login page heading', () => {
    renderApp();

    expect(screen.getByRole('heading', { name: /sign in to your account/i })).toBeTruthy();
  });

  it('signs in an existing user and redirects to the dashboard', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      status: 200,
    } as Response);

    renderApp();

    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: 'kosta@example.com' },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: 'HelloWorld12344!' },
    });
    fireEvent.click(screen.getByRole('button', { name: /sign in/i }));

    await waitFor(() => expect(fetchSpy).toHaveBeenCalled());

    expect(fetchSpy).toHaveBeenCalledWith(`${import.meta.env.VITE_BACKEND_HOST}/api/auth/login`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        email: 'kosta@example.com',
        password: 'HelloWorld12344!',
      }),
    });

    await waitFor(() => expect(screen.queryByRole('heading', { name: /sign in to your account/i })).toBeNull());

    fetchSpy.mockRestore();
  });

  it('registers a new user, signs them in, and redirects to the dashboard', async () => {
    const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue({
      ok: true,
      status: 201,
    } as Response);

    renderApp();

    fireEvent.change(screen.getByLabelText(/email/i), {
      target: { value: 'kosta@example.com' },
    });
    fireEvent.change(screen.getByLabelText(/password/i), {
      target: { value: 'HelloWorld12344!' },
    });
    fireEvent.click(screen.getByRole('button', { name: /sign up/i }));

    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(`${import.meta.env.VITE_BACKEND_HOST}/api/auth/register`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: 'kosta@example.com',
          password: 'HelloWorld12344!',
        }),
      }),
    );

    await waitFor(() =>
      expect(fetchSpy).toHaveBeenCalledWith(`${import.meta.env.VITE_BACKEND_HOST}/api/auth/login`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          email: 'kosta@example.com',
          password: 'HelloWorld12344!',
        }),
      }),
    );

    await waitFor(() => expect(screen.queryByRole('heading', { name: /sign in to your account/i })).toBeNull());

    fetchSpy.mockRestore();
  });
});
