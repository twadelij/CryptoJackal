import { useState } from 'react';
import { LogIn, AlertCircle } from 'lucide-react';
import { login } from '../lib/api';

export default function Login() {
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(username, password);
      window.location.href = '/';
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center">
      <div className="card p-8 w-full max-w-md">
        <div className="flex items-center gap-3 mb-6 justify-center">
          <img src="/logo.png" alt="CryptoJackal" className="w-12 h-12 rounded-lg" />
          <h1 className="text-2xl font-bold">CryptoJackal</h1>
        </div>

        <p className="text-gray-400 text-center mb-6">Log in to access your dashboard</p>

        {error && (
          <div className="mb-4 p-3 bg-red-500/20 border border-red-500/50 rounded-lg flex items-center gap-2 text-red-400">
            <AlertCircle size={18} />
            <span className="text-sm">{error}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm text-gray-400 mb-1">Username</label>
            <input
              type="text"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="admin"
              className="input-field"
              required
            />
          </div>

          <div>
            <label className="block text-sm text-gray-400 mb-1">Password</label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="Enter password"
              className="input-field"
              required
            />
          </div>

          <button
            type="submit"
            disabled={loading}
            className="btn-primary w-full flex items-center justify-center gap-2 disabled:opacity-50"
          >
            {loading ? 'Logging in...' : <><LogIn size={18} /> Log In</>}
          </button>
        </form>

        <div className="mt-6 text-center text-sm text-gray-500">
          <p>Default password is set in your <code>.env</code> file</p>
          <p className="mt-1">Check <code>ADMIN_PASSWORD</code></p>
        </div>
      </div>
    </div>
  );
}
