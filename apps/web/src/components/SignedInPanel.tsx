import { useCallback, useEffect, useState } from "react";
import { ApiError, api, type Me, type Member } from "../api";

// Signed-in home: current user, organization, role, the member list,
// procurement requests, and sign-out. Procurement requests land with
// Milestone 2 part 2.

interface Props {
  me: Me;
  onSignedOut: () => void;
}

type MembersState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "loaded"; members: Member[] };

type RequestState =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "loaded"; request: { id: string; title: string; currency: string; status: string; created_at: string }; quotations: { id: string; vendor_name: string; original_filename: string; status: string; created_at: string }[] };

export default function SignedInPanel({ me, onSignedOut }: Props) {
  const [membersState, setMembersState] = useState<MembersState>({ kind: "loading" });
  const [signingOut, setSigningOut] = useState(false);
  const [requestState, setRequestState] = useState<RequestState>({ kind: "idle" });

  const loadMembers = useCallback(async () => {
    setMembersState({ kind: "loading" });
    try {
      const response = await api.members();
      setMembersState({ kind: "loaded", members: response.members });
    } catch (error) {
      setMembersState({
        kind: "error",
        message: error instanceof ApiError ? error.message : "Unexpected error.",
      });
    }
  }, []);

  useEffect(() => {
    void loadMembers();
  }, [loadMembers]);

  const signOut = async () => {
    setSigningOut(true);
    try {
      await api.logout();
    } catch {
      // Session refresh (onSignedOut) reveals whether it is still
      // valid; nothing sensitive is shown either way.
    } finally {
      setSigningOut(false);
      onSignedOut();
    }
  };

  const createRequest = useCallback(async () => {
    setRequestState({ kind: "loading" });
    try {
      const response = await api.createProcurementRequest({ title: "New request", currency: "USD" });
      setRequestState({
        kind: "loaded",
        request: { id: response.id, title: "New request", currency: "USD", status: "draft", created_at: new Date().toISOString() },
        quotations: [],
      });
    } catch (error) {
      setRequestState({
        kind: "error",
        message: error instanceof ApiError ? error.message : "Unexpected error.",
      });
    }
  }, []);

  const loadRequests = useCallback(async () => {
    setRequestState({ kind: "loading" });
    try {
      const requests = await api.listProcurementRequests();

      if (requests.length === 0) {
        setRequestState({
          kind: "loaded",
          request: { id: "", title: "", currency: "", status: "", created_at: "" },
          quotations: [],
        });
        return;
      }

      const requestsWithQuotes = [] as Array<{ id: string; title: string; currency: string; status: string; created_at: string; quotations: { id: string; vendor_name: string; original_filename: string; status: string; created_at: string }[] }>;

      for (const req of requests) {
        try {
          const quotations = await api.listQuotations(req.id);
          requestsWithQuotes.push({ ...req, quotations });
        } catch {
          // Keep the request visible even if quotation metadata fails to load.
          requestsWithQuotes.push({ ...req, quotations: [] });
        }
      }

      const selectedRequest = requestsWithQuotes[0];
      setRequestState({
        kind: "loaded",
        request: selectedRequest ?? { id: "", title: "", currency: "", status: "", created_at: "" },
        quotations: selectedRequest?.quotations ?? [],
      });
    } catch (error) {
      setRequestState({
        kind: "error",
        message: error instanceof ApiError ? error.message : "Unexpected error.",
      });
    }
  }, []);

  useEffect(() => {
    void loadRequests();
  }, [loadRequests]);

  return (
    <>
      <section className="panel" aria-label="Your account">
        <h2>Signed in</h2>
        <p>
          <strong>{me.user.email}</strong> — {me.organization.name}{" "}
          <span className="muted">({me.role})</span>
        </p>
        <button onClick={() => void signOut()} disabled={signingOut}>
          {signingOut ? "Signing out…" : "Sign out"}
        </button>
      </section>

      <section className="panel" aria-label="Organization members">
        <h2>Members of {me.organization.name}</h2>
        {membersState.kind === "loading" && (
          <p role="status" className="muted">
            Loading members…
          </p>
        )}
        {membersState.kind === "error" && (
          <p role="alert" className="error">
            Could not load members: {membersState.message}{" "}
            <button onClick={() => void loadMembers()}>Retry</button>
          </p>
        )}
        {membersState.kind === "loaded" && membersState.members.length === 0 && (
          <p className="muted">No members yet.</p>
        )}
        {membersState.kind === "loaded" && membersState.members.length > 0 && (
          <table>
            <thead>
              <tr>
                <th>Email</th>
                <th>Role</th>
                <th>Joined</th>
              </tr>
            </thead>
            <tbody>
              {membersState.members.map((member) => (
                <tr key={member.user_id}>
                  <td>{member.email}</td>
                  <td>{member.role}</td>
                  <td>{new Date(member.joined_at).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>

      {/* Create procurement request */}
      {requestState.kind === "idle" && (
        <section className="panel" aria-label="Create procurement request">
          <h2>Create procurement request</h2>
          <p>Add a new purchase request under your organization.</p>
          <button onClick={() => void createRequest()}>Create request</button>
        </section>
      )}

      {requestState.kind === "loading" && (
        <section className="panel">
          <h2>Creating request…</h2>
          <p>Your procurement request is being created.</p>
        </section>
      )}

      {requestState.kind === "error" && (
        <section className="panel" aria-label="Error">
          <h2>Error</h2>
          <p>{requestState.message}</p>
        </section>
      )}

      {requestState.kind === "loaded" && (
        <section className="panel" aria-label="Procurement request detail">
          <h2>Procurement request</h2>
          <p>
            <strong>Title:</strong> {requestState.request.title}<br />
            <strong>Currency:</strong> {requestState.request.currency}<br />
            <strong>Status:</strong> {requestState.request.status}<br />
            <strong>Created:</strong> {requestState.request.created_at}
          </p>

          <h3>Quotations ({requestState.quotations.length})</h3>
          {requestState.quotations.length === 0 && (
            <p>No quotations yet.</p>
          )}
          {requestState.quotations.length > 0 && (
            <ul>
              {requestState.quotations.map((q) => (
                <li key={q.id}>
                  <strong>{q.vendor_name}</strong> — {q.original_filename} — {q.status}
                </li>
              ))}
            </ul>
          )}

          <button onClick={() => void loadRequests()}>Refresh quotations</button>
        </section>
      )}
    </>
  );
}
