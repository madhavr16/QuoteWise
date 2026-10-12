import { useCallback, useEffect, useState } from "react";
import { api, type Health } from "../api";

// Dev status panel: shows the Go API's liveness and readiness through
// the API itself (the browser never calls the AI service directly).

type State =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "loaded"; health: Health; ready: Health };

export default function ServiceStatus() {
  const [state, setState] = useState<State>({ kind: "loading" });

  const refresh = useCallback(async () => {
    setState({ kind: "loading" });
    try {
      const [health, ready] = await Promise.all([api.health(), api.ready()]);
      setState({ kind: "loaded", health, ready });
    } catch (error) {
      setState({
        kind: "error",
        message: error instanceof Error ? error.message : "Unexpected error",
      });
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  return (
    <section className="panel" aria-label="Go API status">
      <h2>Go API</h2>
      {state.kind === "loading" && (
        <p role="status" className="muted">
          Loading service status…
        </p>
      )}
      {state.kind === "error" && (
        <p role="alert" className="error">
          Could not reach the Go API: {state.message}
        </p>
      )}
      {state.kind === "loaded" && (
        <>
          <p>
            liveness: <strong>{state.health.status}</strong> (version {state.health.version})
          </p>
          <h3>Readiness checks (via /readyz)</h3>
          <ul>
            {Object.entries(state.ready.checks ?? {}).map(([name, check]) => (
              <li key={name}>
                {name}: <strong>{check.status}</strong>
                {check.detail ? <span> — {check.detail}</span> : null}
              </li>
            ))}
          </ul>
        </>
      )}
      <button onClick={() => void refresh()}>Refresh status</button>
    </section>
  );
}
