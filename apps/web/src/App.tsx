import { useCallback, useEffect, useState } from "react";
import { ApiError, api, type Me } from "./api";
import AuthPanel from "./components/AuthPanel";
import ServiceStatus from "./components/ServiceStatus";
import SignedInPanel from "./components/SignedInPanel";

// App shell: resolves the session once on load, then shows the
// signed-in workspace or the sign-in panel. Procurement features
// arrive with Milestone 2 part 2 and beyond (docs/implementation-plan.md).

type SessionState =
  | { kind: "checking" }
  | { kind: "signed-out" }
  | { kind: "error"; message: string }
  | { kind: "signed-in"; me: Me };

export default function App() {
  const [session, setSession] = useState<SessionState>({ kind: "checking" });

  const refreshSession = useCallback(async () => {
    setSession({ kind: "checking" });
    try {
      const me = await api.me();
      setSession({ kind: "signed-in", me });
    } catch (error) {
      if (error instanceof ApiError && error.status === 401) {
        setSession({ kind: "signed-out" });
        return;
      }
      setSession({
        kind: "error",
        message: error instanceof Error ? error.message : "Unexpected error",
      });
    }
  }, []);

  useEffect(() => {
    void refreshSession();
  }, [refreshSession]);

  return (
    <main>
      <h1>QuoteWise</h1>
      <p className="muted">
        AI-assisted procurement and quotation intelligence — under
        construction; see <code>docs/implementation-plan.md</code>.
      </p>

      {session.kind === "checking" && (
        <p role="status" className="muted">
          Checking your session…
        </p>
      )}

      {session.kind === "error" && (
        <p role="alert" className="error">
          Could not reach the Go API: {session.message}
        </p>
      )}

      {session.kind === "signed-out" && (
        <>
          <AuthPanel onAuthenticated={() => void refreshSession()} />
          <ServiceStatus />
        </>
      )}

      {session.kind === "signed-in" && (
        <SignedInPanel me={session.me} onSignedOut={() => void refreshSession()} />
      )}
    </main>
  );
}
