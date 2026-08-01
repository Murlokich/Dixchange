const platformChoices = [
  'TypeScript-first codebase',
  'Vite build pipeline',
  'ESLint + Prettier guardrails',
  'Vitest + RTL test harness',
  'Realtime-friendly client layer for SSE/WebSocket feeds',
];

export default function App() {
  return (
    <main className="app-shell">
      <section className="hero-card">
        <p className="eyebrow">Frontend foundation</p>
        <h1>Trading platform UI scaffold</h1>
        <p className="lede">
          This first iteration establishes the tooling and conventions that future features will
          build on. The goal is a small, explicit surface area that remains easy for humans and
          agents to navigate.
        </p>

        <div className="choice-panel">
          <h2>Why this stack</h2>
          <ul>
            {platformChoices.map((choice) => (
              <li key={choice}>{choice}</li>
            ))}
          </ul>
        </div>
      </section>
    </main>
  );
}