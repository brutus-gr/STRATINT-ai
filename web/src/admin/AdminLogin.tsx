import { useState } from 'react';
import { Terminal, Lock } from 'lucide-react';

interface AdminLoginProps {
  onLogin: (password: string) => Promise<{ success: boolean; error?: string }>;
}

export function AdminLogin({ onLogin }: AdminLoginProps) {
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!password) {
      setError('PASSWORD REQUIRED');
      return;
    }

    setIsLoading(true);
    setError('');

    const result = await onLogin(password);

    if (!result.success) {
      setError(result.error || 'LOGIN FAILED');
    }

    setIsLoading(false);
  };

  return (
    <div className="min-h-screen bg-void text-chalk flex items-center justify-center p-6">
      {/* Scan line effect */}
      <div className="scan-line" />
      
      <div className="w-full max-w-md">
        {/* Logo */}
        <div className="text-center mb-8">
          <div className="flex items-center justify-center gap-3 mb-4">
            <Terminal className="w-8 h-8 text-terminal animate-pulse-slow" />
            <h1 className="font-display font-black text-3xl text-white">STRATINT</h1>
          </div>
          <p className="text-sm font-mono text-smoke">ADMINISTRATIVE ACCESS</p>
        </div>

        {/* Login Form */}
        <div className="border-2 border-terminal bg-concrete">
          {/* Header */}
          <div className="px-6 py-4 border-b-2 border-terminal bg-terminal/10">
            <div className="flex items-center gap-3">
              <Lock className="w-5 h-5 text-terminal" />
              <h2 className="font-display font-black text-base text-terminal tracking-tight">
                AUTHENTICATION REQUIRED
              </h2>
            </div>
          </div>

          {/* Form */}
          <form onSubmit={handleSubmit} className="p-6 space-y-6">
            <div className="space-y-2">
              <label className="block text-xs font-mono text-smoke font-medium">
                ADMIN PASSWORD
              </label>
              <input
                type="password"
                value={password}
                onChange={(e) => {
                  setPassword(e.target.value);
                  setError('');
                }}
                className="w-full px-4 py-3 bg-void border-2 border-steel text-chalk font-mono focus:border-terminal focus:outline-none transition-colors"
                placeholder="Enter password..."
                autoFocus
              />
              {error && (
                <p className="text-xs font-mono text-threat-critical flex items-center gap-2">
                  <span>⚠</span> {error}
                </p>
              )}
            </div>

            <button
              type="submit"
              disabled={isLoading}
              className="w-full px-6 py-4 border-2 border-terminal bg-terminal/10 text-terminal hover:bg-terminal hover:text-void transition-all font-mono text-sm font-bold disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? '[AUTHENTICATING...]' : '[AUTHENTICATE]'}
            </button>

            <div className="pt-4 border-t border-steel">
              <p className="text-xs font-mono text-fog text-center">
                Unauthorized access is logged and monitored
              </p>
            </div>
          </form>
        </div>
      </div>
    </div>
  );
}
