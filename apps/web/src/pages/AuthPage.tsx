import { useMemo, useState } from "react";
import { useAuth } from "../auth/AuthProvider";

export default function AuthPage() {
  const { signup, login } = useAuth();
  const mode = useMemo(() => (new URLSearchParams(location.hash.split("?")[1]).get("mode") === "signup" ? "signup" : "login"), []);
  const [username, setU] = useState("");
  const [password, setP] = useState("");
  const [err, setErr] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr(null); setLoading(true);
    try {
      if (mode === "signup") await signup(username, password);
      else await login(username, password);
      location.hash = "#/profile"; // after auth, show profile (stats)
    } catch (e: any) {
      setErr(e?.message ?? "Authentication failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="p-4 max-w-sm mx-auto">
      <h1 className="text-xl font-bold mb-3">{mode === "signup" ? "Create account" : "Sign in"}</h1>
      {err && <div className="text-red-600 text-sm mb-2">{err}</div>}
      <form onSubmit={onSubmit} className="flex flex-col gap-2">
        <input className="border p-2 rounded" placeholder="username" value={username} onChange={e=>setU(e.target.value)} />
        <input className="border p-2 rounded" placeholder="password" type="password" value={password} onChange={e=>setP(e.target.value)} />
        <button disabled={loading} className="bg-black text-white rounded p-2">{loading ? "…" : (mode==="signup"?"Sign up":"Sign in")}</button>
      </form>
      <div className="mt-3 text-sm">
        {mode==="signup" ? <a href="#/auth">Already have an account? Sign in</a> : <a href="#/auth?mode=signup">Create an account</a>}
      </div>
    </div>
  );
}
