import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import App from './App';

describe('App', () => {
  it('renders the login page heading', () => {
    render(<App />);

    expect(screen.getByRole('heading', { name: /sign in to your account/i })).toBeTruthy();
  });
});