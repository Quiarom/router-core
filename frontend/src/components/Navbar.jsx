import React from "react";
import { Database, Radio } from "lucide-react";

export function Navbar({ isLive, setIsLive, wanStatus: _wanStatus = "connected", activeTab = "assistant", setActiveTab }) {
  return (
    <header className="sticky top-0 z-50 border-b-2 border-neutral-800 bg-black">
      <div className="w-full flex flex-col md:flex-row items-center justify-between px-4 sm:px-6 lg:px-8 py-3 gap-3 relative">
        {/* Brand */}
        <div className="flex items-center self-start md:self-auto py-1">
          <span className="font-bold text-white tracking-tight font-mono text-base sm:text-lg">
            router-core
          </span>
        </div>

        {/* Main Navigation Links (Center Aligned) */}
        <nav className="flex items-center justify-center gap-6 sm:gap-10 font-mono text-base sm:text-lg md:absolute md:left-1/2 md:-translate-x-1/2">
          <button
            onClick={() => setActiveTab("assistant")}
            className={`font-black uppercase tracking-wider transition-colors cursor-pointer ${
              activeTab === "assistant"
                ? "text-primary"
                : "text-neutral-400 hover:text-white"
            }`}
          >
            Asistente
          </button>
          <button
            onClick={() => setActiveTab("telemetry")}
            className={`font-black uppercase tracking-wider transition-colors cursor-pointer ${
              activeTab === "telemetry"
                ? "text-primary"
                : "text-neutral-400 hover:text-white"
            }`}
          >
            Estadísticas
          </button>
        </nav>

        {/* Right actions */}
        <div className="flex items-center gap-3">
          {/* Data source switch */}
          <button
            onClick={() => setIsLive(!isLive)}
            className={`flex items-center gap-2 px-3 py-1.5 border-2 text-xs font-mono font-bold uppercase transition-colors cursor-pointer ${
              isLive
                ? "border-primary bg-primary text-white"
                : "border-neutral-700 bg-neutral-900 text-neutral-300 hover:border-neutral-500"
            }`}
            title="Alternar entre fixtures de prueba y router-core serve local"
          >
            {isLive ? (
              <>
                <Radio className="h-3.5 w-3.5 text-white animate-pulse" />
                <span>Live (127.0.0.1:8484)</span>
              </>
            ) : (
              <>
                <Database className="h-3.5 w-3.5 text-primary" />
                <span>Mock Fixtures</span>
              </>
            )}
          </button>

          <a
            href="https://github.com/Quiarom/router-core"
            target="_blank"
            rel="noreferrer"
            className="flex h-9 w-9 items-center justify-center border-2 border-neutral-800 bg-neutral-900 text-neutral-300 hover:bg-neutral-800 hover:text-white hover:border-neutral-600 transition-colors"
            title="Ver repositorio en GitHub"
          >
            <svg className="h-4 w-4 fill-current" viewBox="0 0 24 24">
              <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/>
            </svg>
          </a>
        </div>
      </div>
    </header>
  );
}

export default Navbar;
