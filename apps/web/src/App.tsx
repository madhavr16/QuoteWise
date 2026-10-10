import { useCallback, useEffect, useState } from "react";

// Minimal Milestone 1 page: shows the local service status through the
// Go API (the browser never calls the AI service directly). Real pages
// (dashboard, requests, upload, review, comparison, Q&A, report) arrive
// in later milestones.

type CheckResult = { status: string; detail?: string };

type Health = {
  status: string;
  version: string;
  checks?: Record<string, CheckResult>;
};

type LoadState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "loaded"; health: Health; readiness: Health };

async function fetchJson<T>(path: string): Promise<T> {
  const response = await fetch(path, { headers: { Accept: "application/json" } });
  if (!response.ok) {
    throw new Error(`${path} returned HTTP ${response.status}`);
  }
  return (await response.json()) as T;
}

export default function App() {
  const [loadState, setLoadState] = useState<LoadState>({ kind: "loading" });

  const refresh = useCallback(async () => {
    setLoadState({ kind: "loading" });
    try {
      const [health, readiness] = await Promise.all([
        fetchJson<Health>("/healthz"),
        fetchJson<Health>("/readyz"),
      ]);
      setLoadState({ kind: "loaded", health, readiness });
    } catch (error) {
      setLoadState({
        kind: "error",
        message: error instanceof Error ? error.message : "Unexpected error",
      });
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <main>
      <h1>QuoteWise</h1>
      <p>
        Development status page — the product UI (requests, uploads, review,
        comparison, Q&amp;A, reports) is built from Milestone 2 onwards. See{" "}
        <code>docs/implementation-plan.md</code>.
      </p>
      <button onClick={() => void refresh()}>Refresh status</button>

      {loadState.kind === "loading" && <p role="status">Loading service status…</p>}

      {loadState.kind === "error" && (
        <p role="alert" className="error">
          Could not reach the Go API: {loadState.message}
        </p>
      )}

      {loadState.kind === "loaded" && (
        <section aria-label="Go API status">
          <h2>Go API</h2>
          <p>
            liveness: <strong>{loadState.health.status}</strong> (version{" "}
            {loadState.health.version})
          </p>
          <h3>Readiness checks (via /readyz)</h3>
          <ul>
            {Object.entries(loadState.readiness.checks ?? {}).map(([name, check]) => (
              <li key={name}>
                {name}: <strong>{check.status}</strong>
                {check.detail ? <span> — {check.detail}</span> : null}
              </li>
            ))}
          </ul>
        </section>
      )}
    </main>
  );
}
