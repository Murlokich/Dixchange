import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { storeEmail } from '../lib/session';

type AuthPayload = {
  email: string;
  password: string;
};

type StatusState = {
  kind: 'idle' | 'success' | 'error';
  message: string;
};

const registerPath = '/api/auth/register';
const loginPath = '/api/auth/login';
const dashboardPath = '/dashboard';

export default function LoginPage() {
  const navigate = useNavigate();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [status, setStatus] = useState<StatusState>({ kind: 'idle', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [isLoggingIn, setIsLoggingIn] = useState(false);

  const getBackendHost = () => {
    const backendHost = import.meta.env.VITE_BACKEND_HOST?.trim();

    if (!backendHost) {
      setStatus({
        kind: 'error',
        message: 'Set VITE_BACKEND_HOST in envs/.env before continuing.',
      });
      return null;
    }

    return backendHost;
  };

  const requestLogin = async (backendHost: string, payload: AuthPayload) => {
    const response = await fetch(`${backendHost}${loginPath}`, {
      method: 'POST',
      credentials: 'include',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(payload),
    });

    if (!response.ok) {
      throw new Error(`Login request failed with status ${response.status}`);
    }
  };

  const handleLogin = async () => {
    const backendHost = getBackendHost();
    if (!backendHost) {
      return;
    }

    setIsLoggingIn(true);
    setStatus({ kind: 'idle', message: '' });

    try {
      await requestLogin(backendHost, { email, password });
      storeEmail(email);
      navigate(dashboardPath);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to complete sign in request.';

      setStatus({
        kind: 'error',
        message,
      });
    } finally {
      setIsLoggingIn(false);
    }
  };

  const handleSignUp = async () => {
    const backendHost = getBackendHost();
    if (!backendHost) {
      return;
    }

    const payload: AuthPayload = {
      email,
      password,
    };

    setIsSubmitting(true);
    setStatus({ kind: 'idle', message: '' });

    try {
      const response = await fetch(`${backendHost}${registerPath}`, {
        method: 'POST',
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`Register request failed with status ${response.status}`);
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to complete sign up request.';

      setStatus({
        kind: 'error',
        message,
      });
      setIsSubmitting(false);
      return;
    }

    setIsLoggingIn(true);

    try {
      await requestLogin(backendHost, payload);
      storeEmail(email);
      navigate(dashboardPath);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to complete sign in request.';

      setStatus({
        kind: 'error',
        message: `Account created, but automatic sign in failed: ${message}. Please sign in manually.`,
      });
    } finally {
      setIsSubmitting(false);
      setIsLoggingIn(false);
    }
  };

  return (
    <main className="auth-page">
      <section className="auth-panel" aria-labelledby="auth-title">
        <div className="auth-intro">
          <p className="eyebrow">Trading platform</p>
          <h1 id="auth-title">Sign in to your account</h1>
          <p className="lede">
            Use your email and password to enter the platform. Sign up is available for users who
            do not have an account yet.
          </p>

          <div className="auth-badges" aria-label="Platform highlights">
            <span>Secure access</span>
            <span>Realtime-ready</span>
            <span>Modern UI</span>
          </div>
        </div>

        <form
          className="auth-card"
          onSubmit={(event) => {
            event.preventDefault();
            void handleLogin();
          }}
        >
          <div className="field-group">
            <label htmlFor="email">Email</label>
            <input
              id="email"
              name="email"
              type="email"
              placeholder="you@company.com"
              value={email}
              onChange={(event) => setEmail(event.target.value)}
            />
          </div>

          <div className="field-group">
            <label htmlFor="password">Password</label>
            <input
              id="password"
              name="password"
              type="password"
              placeholder="Enter your password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
            />
          </div>

          <button type="submit" className="primary-action" disabled={isLoggingIn || isSubmitting}>
            {isLoggingIn ? 'Signing in...' : 'Sign in'}
          </button>

          <button
            type="button"
            className="secondary-action"
            onClick={handleSignUp}
            disabled={isSubmitting || isLoggingIn}
          >
            {isSubmitting ? 'Signing up...' : 'Sign up'}
          </button>

          <p className={`form-note ${status.kind}`} aria-live="polite">
            {status.message || 'Backend host is read from envs/.env and used for the auth requests.'}
          </p>
        </form>
      </section>
    </main>
  );
}
