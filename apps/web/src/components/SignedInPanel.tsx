import { useCallback, useEffect, useState } from "react";
import { ApiError, api, type Me, type Member } from "../api";

// Signed-in home: current user, organization, role, the member list,
// and sign-out. Procurement requests land with Milestone 2 part 2.

interface Props {
  me: Me;
  onSignedOut: () => void;
}

type MembersState =
  | { kind: "loading" }
  | { kind: "error"; message: string }
  | { kind: "loaded"; members: Member[] };

export default function SignedInPanel({ me, onSignedOut }: Props) {
  const [membersState, setMembersState] = useState<MembersState>({ kind: "loading" });
  const [signingOut, setSigningOut] = useState(false);

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
    </>
  );
}
