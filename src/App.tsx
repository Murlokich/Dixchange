import { useState } from 'react';

type RegisterPayload = {
  email: string;
  password: string;
};

type StatusState = {
  kind: 'idle' | 'success' | 'error';
  message: string;
};

const registerPath = '/api/auth/register';

export default function App() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [status, setStatus] = useState<StatusState>({ kind: 'idle', message: '' });
  const [isSubmitting, setIsSubmitting] = useState(false);

  const handleSignUp = async () => {
    const backendHost = import.meta.env.VITE_BACKEND_HOST?.trim();

    if (!backendHost) {
      setStatus({
        kind: 'error',
        message: 'Set VITE_BACKEND_HOST in envs/.env before using sign up.',
      });
      return;
    }

    const payload: RegisterPayload = {
      email,
      password,
    };

    setIsSubmitting(true);
    setStatus({ kind: 'idle', message: '' });

    try {
      const response = await fetch(`${backendHost}${registerPath}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        throw new Error(`Register request failed with status ${response.status}`);
      }

      setStatus({
        kind: 'success',
        message: 'Sign up request sent successfully.',
      });
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unable to complete sign up request.';

      setStatus({
        kind: 'error',
        message,
      });
    } finally {
      setIsSubmitting(false);
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

        <form className="auth-card" onSubmit={(event) => event.preventDefault()}>
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

          <button type="submit" className="primary-action">
            Sign in
          </button>

          <button type="button" className="secondary-action" onClick={handleSignUp} disabled={isSubmitting}>
            {isSubmitting ? 'Signing up...' : 'Sign up'}
          </button>

          <p className={`form-note ${status.kind}`} aria-live="polite">
            {status.message || 'Backend host is read from envs/.env and used for the register request.'}
          </p>
        </form>
      </section>
    </main>
  );
}