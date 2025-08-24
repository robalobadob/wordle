import { useAuth } from "../auth/AuthProvider";

export default function Header() {
  const { me, logout } = useAuth();
  return (
    <header className="flex items-center justify-between p-3 border-b">
      <a href="#" className="font-bold">Wordle</a>
      <nav className="flex items-center gap-3">
        {me ? (
          <>
            <span className="text-sm opacity-70">Hello, {me.username}</span>
            <a className="text-sm underline" href="#/profile">Profile</a>
            <button className="text-sm" onClick={logout}>Log out</button>
          </>
        ) : (
          <>
            <a className="text-sm underline" href="#/auth">Sign in</a>
            <a className="text-sm underline" href="#/auth?mode=signup">Sign up</a>
          </>
        )}
      </nav>
    </header>
  );
}
