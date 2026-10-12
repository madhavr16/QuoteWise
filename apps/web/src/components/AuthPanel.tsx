import { useState, type FormEvent } from "react";
import { ApiError, api } from "../api";

// Sign-in / registration panel. Client-side validation mirrors the
// server's domain rules for fast feedback; the server re-validates
// everything (defense in depth).

interface Props {
  onAuthenticated: () => void;
}

type Mode = "login" | "register";

const EMAIL_PATTERN = /^[^@\s]+@[^@\s]+\.[^@\s]+$/;

export default function AuthPanel({ onAuthenticated }: Props) {
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [organizationName, setOrganizationName] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const switchMode = (next: Mode) => {
    setMode(next);
    setFieldErrors({});
    setError(null);
  };

  const validate = (): boolean => {
    const errors: Record<string, string> = {};
    if (!EMAIL_PATTERN.test(email.trim())) {
      errors.email = "Enter a valid email address.";
    }
    if (password.length < 12) {
      errors.password = "Password must be at least 12 characters.";
    } else if (password.length > 128) {
      errors.password = "Password must be at most 128 characters.";
    }
    if (mode === "register") {
      const name = organizationName.trim();
      if (name.length < 1) {
        errors.organization_name = "Organization name is required.";
      } else if (name.length > 200) {
        errors.organization_name = "Organization name must be at most 200 characters.";
      }
    }
    setFieldErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    if (!validate()) {
      return;
    }
    setSubmitting(true);
    try {
      if (mode === "register") {
        await api.register({
          email: email.trim(),
          password,
          organization_name: organizationName.trim(),
        });
      } else {
        await api.login(email.trim(), password);
      }
      onAuthenticated();
    } catch (err) {
      setError(
        err instanceof ApiError ? err.message : "Unexpected error. Please try again.",
      );
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <section className="panel" aria-label={mode === "login" ? "Sign in" : "Create an account"}>
      <h2>{mode === "login" ? "Sign in" : "Create an account"}</h2>
      <p className="muted">
        {mode === "login"
          ? "Sign in to manage your organization."
          : "Registration creates your organization and makes you its owner."}
      </p>
      <form onSubmit={(event) => void submit(event)} noValidate>
        <label htmlFor="email">Email</label>
        <input
          id="email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          autoComplete="email"
        />
        {fieldErrors.email && (
          <p role="alert" className="field-error">
            {fieldErrors.email}
          </p>
        )}

        <label htmlFor="password">Password</label>
        <input
          id="password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          autoComplete={mode === "login" ? "current-password" : "new-password"}
        />
        {fieldErrors.password && (
          <p role="alert" className="field-error">
            {fieldErrors.password}
          </p>
        )}

        {mode === "register" && (
          <>
            <label htmlFor="organization-name">Organization name</label>
            <input
              id="organization-name"
              value={organizationName}
              onChange={(event) => setOrganizationName(event.target.value)}
            />
            {fieldErrors.organization_name && (
              <p role="alert" className="field-error">
                {fieldErrors.organization_name}
              </p>
            )}
          </>
        )}

        {error && (
          <p role="alert" className="error">
            {error}
          </p>
        )}

        <button type="submit" disabled={submitting}>
          {submitting
            ? "Please wait…"
            : mode === "login"
              ? "Sign in"
              : "Create account"}
        </button>
      </form>
      <p className="muted">
        {mode === "login" ? "No account yet? " : "Already registered? "}
        <button
          type="button"
          className="link"
          onClick={() => switchMode(mode === "login" ? "register" : "login")}
        >
          {mode === "login" ? "Create one" : "Sign in instead"}
        </button>
      </p>
    </section>
  );
}
