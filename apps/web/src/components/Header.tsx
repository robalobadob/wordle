import { useAuth } from "../auth/AuthProvider";

export default function Header() {
  const { me, logout } = useAuth();
  return (
    <header className="site-header">
      <a href="#" className="brand">Wordle</a>

      <nav className="nav">
        {me ? (
          <>
            <span className="text-sm muted">Hello, {me.username}</span>
            <a className="text-sm link-underline" href="#/profile">Profile</a>
            <button className="text-sm btn-link link-underline" onClick={logout}>Log out</button>
          </>
        ) : (
          <>
            <a className="text-sm link-underline" href="#/auth">Sign in</a>
            <a className="text-sm link-underline" href="#/auth?mode=signup">Sign up</a>
          </>
        )}
      </nav>
    </header>
  );
}
