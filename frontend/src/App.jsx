import React, { useState } from "react";

import { AiAssistantView } from "@/components/AiAssistantView";
import { Dashboard } from "@/components/Dashboard";
import { Navbar } from "@/components/Navbar";
import { RouterSetup } from "@/components/RouterSetup";
import { disconnectRouter, isDesktopRuntime } from "@/lib/desktop";

export function App() {
  const isDesktop = isDesktopRuntime();
  const [activeTab, setActiveTab] = useState("assistant");
  const [isLive, setIsLive] = useState(false);
  const [wanStatus, setWanStatus] = useState("unknown");
  const [connection, setConnection] = useState(null);

  const handleConnected = (nextConnection) => {
    setConnection(nextConnection);
    setIsLive(true);
    setActiveTab("telemetry");
  };

  const handleDisconnect = async () => {
    await disconnectRouter();
    setConnection(null);
    setIsLive(false);
    setWanStatus("unknown");
  };

  if (isDesktop && !connection) {
    return <RouterSetup onConnected={handleConnected} />;
  }

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-primary selection:text-white">
      <Navbar
        isLive={isLive}
        setIsLive={setIsLive}
        isDesktop={isDesktop}
        connection={connection}
        onDisconnect={handleDisconnect}
        wanStatus={wanStatus}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
      />

      <main className="flex-1">
        {activeTab === "assistant" ? (
          <AiAssistantView isLive={isLive} />
        ) : (
          <Dashboard
            isLive={isLive}
            onWanStatusChange={setWanStatus}
          />
        )}
      </main>

      <footer className="border-t-2 border-neutral-800 bg-black py-4 text-xs text-neutral-400 font-mono">
        <div className="w-full px-4 sm:px-6 lg:px-8 flex items-center justify-center text-center">
          <div className="flex flex-wrap items-center justify-center gap-2">
            <span>GMI Cloud × MiniMax Week Hackathon</span>
            <span className="text-neutral-600">|</span>
            <a
              href="https://x.com/QuiaromDev"
              target="_blank"
              rel="noreferrer"
              className="text-white hover:text-primary font-bold transition-colors underline decoration-neutral-700 underline-offset-4 hover:decoration-primary"
            >
              QuiaromDev
            </a>
            <a
              href="https://x.com/MatiasBlnc"
              target="_blank"
              rel="noreferrer"
              className="text-white hover:text-primary font-bold transition-colors underline decoration-neutral-700 underline-offset-4 hover:decoration-primary"
            >
              MatiasBlnc
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
