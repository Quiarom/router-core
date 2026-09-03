import React, { useState } from "react";
import { Navbar } from "@/components/Navbar";
import { AiAssistantView } from "@/components/AiAssistantView";
import { Dashboard } from "@/components/Dashboard";

export function App() {
  const [activeTab, setActiveTab] = useState("assistant");
  const [isLive, setIsLive] = useState(false);
  const [wanStatus, setWanStatus] = useState("connected");

  return (
    <div className="min-h-screen bg-black text-white flex flex-col font-sans selection:bg-primary selection:text-white">
      {/* Top Navbar */}
      <Navbar
        isLive={isLive}
        setIsLive={setIsLive}
        wanStatus={wanStatus}
        activeTab={activeTab}
        setActiveTab={setActiveTab}
      />

      {/* Main Content View */}
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

      {/* Global Brutalist Footer */}
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
