export default function App() {
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

        <form className="auth-card">
          <div className="field-group">
            <label htmlFor="email">Email</label>
            <input id="email" name="email" type="email" placeholder="you@company.com" />
          </div>

          <div className="field-group">
            <label htmlFor="password">Password</label>
            <input id="password" name="password" type="password" placeholder="Enter your password" />
          </div>

          <button type="submit" className="primary-action">
            Sign in
          </button>

          <button type="button" className="secondary-action">
            Sign up
          </button>

          <p className="form-note">
            Backend endpoints and contract wiring will be added in the next iteration.
          </p>
        </form>
      </section>
    </main>
  );
}