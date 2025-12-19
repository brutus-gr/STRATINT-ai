import { Activity } from 'lucide-react';
import { Link } from 'react-router-dom';

export function Header() {
  return (
    <header className="fixed top-0 left-0 right-0 z-50 border-b-2 border-steel bg-void/98 backdrop-blur-md">
      <div className="px-3 md:px-6 py-3 flex items-center justify-between">
        {/* Logo */}
        <Link to="/" className="flex items-center gap-2 md:gap-4">
          <div className="relative">
            <div className="w-3 h-3 bg-terminal rounded-full animate-pulse-slow" />
            <div className="absolute inset-0 w-3 h-3 bg-terminal rounded-full blur-md opacity-75" />
          </div>
          <h1 className="text-base md:text-xl font-display font-black text-white tracking-tight">
            <span className="glitch-text" data-text="STRATINT">STRATINT</span>
          </h1>
          <span className="hidden sm:inline-block text-xs text-smoke font-mono px-2 py-0.5 border border-steel bg-concrete">v1.0</span>
        </Link>

        {/* Live Metrics */}
        <div className="flex items-center gap-2 md:gap-8 text-xs font-mono">
          {/* Live indicator */}
          <div className="flex items-center gap-1 md:gap-2 px-2 md:px-4 py-1.5 border-2 border-terminal bg-terminal/10">
            <Activity className="w-4 h-4 text-terminal animate-pulse-slow" />
            <span className="text-terminal font-bold tracking-wider text-xs">LIVE</span>
          </div>

          {/* Admin Link - Hidden for production */}
          {/* <div className="hidden sm:block h-5 w-px bg-steel" />
          <Link
            to="/admin"
            className="hidden sm:flex items-center gap-2 px-3 md:px-4 py-1.5 border border-steel bg-concrete hover:border-terminal hover:text-terminal transition-all text-xs font-mono font-medium"
          >
            ADMIN
          </Link> */}
        </div>
      </div>
    </header>
  );
}
