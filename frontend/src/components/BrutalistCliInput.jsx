import React, { useState, useEffect } from "react";
import { 
  Terminal, 
  Sparkles, 
  CornerDownLeft, 
  Trash2, 
  History, 
  X, 
  ArrowRight
} from "lucide-react";

export function BrutalistCliInput({ onQuerySuccess }) {
  const [query, setQuery] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [isDrawerOpen, setIsDrawerOpen] = useState(false);
  const [history, setHistory] = useState([
    {
      timestamp: "19:10:02",
      prompt: "¿Está expuesta mi red al exterior?",
      command: "router-core-agent query --security-audit --host 192.168.1.1",
      tools: [
        'get_security("dmz") -> { "state": "verified", "dmz_enabled": "false" }',
        'get_security("forwarding") -> { "state": "verified", "forwarding_rules": [] }'
      ],
      response: "Tu red está totalmente protegida. El router tiene el cerrojo puesto (sin DMZ ni puertos abiertos). Ningún atacante o intruso en internet puede entrar a tus dispositivos.",
      status: "PROTEGIDO"
    }
  ]);

  // Close drawer on Escape key
  useEffect(() => {
    const handleKeyDown = (e) => {
      if (e.key === "Escape") setIsDrawerOpen(false);
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const handleExecute = (customPrompt) => {
    const text = (customPrompt || query).trim();
    if (!text) return;

    setIsProcessing(true);
    setQuery("");

    setTimeout(() => {
      let command = `router-core-agent eval --query "${text.slice(0, 32)}..."`;
      let tools = [];
      let response = "";
      let status = "NORMAL";

      const lower = text.toLowerCase();
      if (lower.includes("peligro") || lower.includes("intruso") || lower.includes("expuest") || lower.includes("segur")) {
        command = "router-core-agent query --inspect-surfaces --audit";
        tools = [
          'get_security("dmz") -> { dmz_enabled: false }',
          'get_security("forwarding") -> { active_rules: 0 }'
        ];
        response = "Diagnóstico de Seguridad: Tu red NO está expuesta. No hay computadoras desprotegidas en la DMZ ni puertos abiertos a desconocidos. Tu conexión está completamente resguardada.";
        status = "PROTEGIDO";
      } else if (lower.includes("lento") || lower.includes("corta") || lower.includes("mejorar") || lower.includes("velocidad")) {
        command = "router-core-agent analyze --throughput --latency-check";
        tools = [
          'get_status() -> { uptime: "5h33m", wanStatus: "connected" }',
          'channel_telemetry() -> { current_ch: 6, interference: "LOW" }'
        ];
        response = "Análisis de Velocidad: La conexión WAN externa está entregando 95 Mbps estables. El canal de Wi-Fi actual está limpio y sin congestión. Si notas lentitud en algún rincón, se debe a la distancia con las paredes.";
        status = "OPTIMIZADO";
      } else if (lower.includes("quién") || lower.includes("quien") || lower.includes("conectad") || lower.includes("usando")) {
        command = "router-core-agent dhcp-leases --resolve-macs";
        tools = [
          'get_clients() -> { count: 3, leases: ["192.168.1.100", "192.168.1.101", "192.168.1.105"] }'
        ];
        response = "Dispositivos en línea: Hay 3 aparatos reconocidos por el router: una laptop, un teléfono móvil y un televisor inteligente. Todos son dispositivos legítimos de la casa.";
        status = "ACTIVO";
      } else {
        command = "router-core-agent inspect --general";
        tools = [
          'get_device() -> { model: "TL-WR841N/ND", fw: "3.15.9" }',
          'get_status() -> { reachable: true }'
        ];
        response = "Información del Router: Equipo TP-Link con firmware oficial verificado. Operación 100% de solo lectura. Todo funciona correctamente.";
        status = "OK";
      }

      const newEntry = {
        timestamp: new Date().toLocaleTimeString("es-AR", { hour: "2-digit", minute: "2-digit", second: "2-digit" }),
        prompt: text,
        command,
        tools,
        response,
        status
      };

      setHistory((prev) => [newEntry, ...prev]);
      setIsProcessing(false);
      if (onQuerySuccess) onQuerySuccess(newEntry);
    }, 600);
  };

  const handleClearHistory = () => {
    setHistory([]);
  };

  const latestEntry = history[0];

  return (
    <>
      <div className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-5 text-white">
        {/* Input principal directo con botón de historial lateral */}
        <div className="space-y-4 pt-1">
          <div className="flex flex-col sm:flex-row gap-2">
            <div className="relative flex-1">
              <Sparkles className="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-primary pointer-events-none" />
              <input
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleExecute()}
                placeholder="¿Cómo puedo ayudarte?"
                className="w-full h-12 bg-black border-2 border-neutral-800 focus:border-primary focus:outline-none pl-11 pr-4 text-sm text-white placeholder-neutral-500 font-sans"
              />
            </div>

            <button
              onClick={() => handleExecute()}
              disabled={isProcessing || !query.trim()}
              className="h-12 px-6 bg-primary hover:bg-primary-hover active:bg-black text-white font-bold uppercase tracking-wider text-xs font-mono transition-colors disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2 border-2 border-primary cursor-pointer shrink-0"
            >
              {isProcessing ? (
                <>
                  <span className="animate-spin">⟳</span>
                  <span>ANALIZANDO...</span>
                </>
              ) : (
                <>
                  <CornerDownLeft className="h-4 w-4" />
                  <span>CONSULTAR</span>
                </>
              )}
            </button>

            <button
              onClick={() => setIsDrawerOpen(true)}
              className="h-12 px-5 bg-neutral-900 hover:bg-neutral-800 border-2 border-neutral-700 text-neutral-300 hover:text-white text-xs font-mono font-bold uppercase transition-colors cursor-pointer flex items-center justify-center gap-2 shrink-0"
              title="Abrir panel lateral con todas las consultas"
            >
              <History className="h-4 w-4 text-primary" />
              <span>HISTORIAL</span>
            </button>
          </div>
        </div>

        {/* Respuesta Inmediata Activa (Última Consulta) */}
        {latestEntry && (
          <div className="border-t-2 border-neutral-800 pt-4 space-y-3">
            <div className="flex items-center justify-between text-xs font-mono text-neutral-400">
              <span className="flex items-center gap-1.5 font-bold uppercase">
                <Terminal className="h-3.5 w-3.5 text-primary" />
                ÚLTIMA RESPUESTA DEL ASISTENTE ({latestEntry.timestamp})
              </span>

              {history.length > 1 && (
                <button
                  onClick={() => setIsDrawerOpen(true)}
                  className="text-primary hover:underline font-mono text-xs flex items-center gap-1 cursor-pointer font-bold"
                >
                  <span>Ver anteriores ({history.length})</span>
                  <ArrowRight className="h-3 w-3" />
                </button>
              )}
            </div>

            <div className="border-l-2 border-primary pl-4 py-2 space-y-2">
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm sm:text-base font-bold text-white font-sans flex items-center gap-2">
                  <Sparkles className="h-4 w-4 text-primary shrink-0" />
                  "{latestEntry.prompt}"
                </span>
                <span className="text-xs font-mono bg-neutral-900 text-neutral-300 border border-neutral-800 px-2 py-0.5 font-bold">
                  {latestEntry.status}
                </span>
              </div>

              <div className="font-mono text-xs text-neutral-400 space-y-0.5 pt-0.5">
                <div className="text-neutral-500">$ {latestEntry.command}</div>
                {latestEntry.tools.map((t, i) => (
                  <div key={i} className="text-primary font-semibold">
                    &gt;&gt; [GET] {t}
                  </div>
                ))}
              </div>

              <p className="text-sm sm:text-base text-neutral-200 leading-relaxed font-sans pt-1">
                {latestEntry.response}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* ========================================================================= */}
      {/* PANEL LATERAL (DRAWER / SLIDE-OVER) PARA CHATS ILIMITADOS */}
      {/* ========================================================================= */}
      {isDrawerOpen && (
        <div className="fixed inset-0 z-50 flex justify-end">
          {/* Backdrop con desenfoque suave */}
          <div 
            className="fixed inset-0 bg-black/80 backdrop-blur-xs transition-opacity"
            onClick={() => setIsDrawerOpen(false)}
          />

          {/* Panel Lateral Deslizable */}
          <aside className="relative z-50 w-full sm:max-w-lg md:max-w-xl h-full bg-neutral-950 border-l-2 border-neutral-800 shadow-2xl flex flex-col text-white font-sans">
            {/* Header del Panel */}
            <div className="p-4 sm:p-5 border-b-2 border-neutral-800 flex items-center justify-between gap-3 bg-black">
              <div className="flex items-center gap-2.5">
                <div className="p-1.5 bg-primary text-white border border-primary">
                  <History className="h-4 w-4" />
                </div>
                <div>
                  <h3 className="text-sm sm:text-base font-black uppercase font-mono tracking-tight text-white">
                    HISTORIAL DE CONSULTAS
                  </h3>
                  <span className="text-xs text-neutral-500 font-mono block">
                    {history.length} {history.length === 1 ? "interacción guardada" : "interacciones guardadas"}
                  </span>
                </div>
              </div>

              <div className="flex items-center gap-2">
                {history.length > 0 && (
                  <button
                    onClick={handleClearHistory}
                    className="px-2.5 py-1 text-xs font-mono border border-neutral-800 hover:border-rose-500 hover:text-rose-400 text-neutral-400 transition-colors cursor-pointer flex items-center gap-1"
                    title="Vaciar todo el historial"
                  >
                    <Trash2 className="h-3 w-3" />
                    <span>Vaciar</span>
                  </button>
                )}

                <button
                  onClick={() => setIsDrawerOpen(false)}
                  className="p-1.5 border border-neutral-800 hover:border-neutral-600 text-neutral-400 hover:text-white transition-colors cursor-pointer"
                  title="Cerrar panel (Esc)"
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            </div>

            {/* Feed Scrollable del Historial (soporta 50+ chats fácilmente) */}
            <div className="flex-1 overflow-y-auto p-4 sm:p-6 space-y-6">
              {history.length === 0 ? (
                <div className="h-full flex flex-col items-center justify-center text-center p-6 text-neutral-500 space-y-2 font-mono text-xs">
                  <Terminal className="h-8 w-8 text-neutral-700" />
                  <p className="uppercase font-bold">No hay consultas en el historial</p>
                  <p className="text-neutral-600 font-sans text-xs">
                    Escribe una pregunta en el input principal para comenzar a auditar el router.
                  </p>
                </div>
              ) : (
                history.map((item, idx) => (
                  <div 
                    key={idx} 
                    className="border-l-2 border-primary pl-4 py-1 space-y-2 bg-neutral-900/40 p-3 border-r border-t border-b border-neutral-800/60"
                  >
                    <div className="flex items-center justify-between gap-2 border-b border-neutral-800 pb-2">
                      <div className="flex items-center gap-1.5 text-xs text-neutral-400 font-mono">
                        <span className="text-primary font-bold">#{history.length - idx}</span>
                        <span>•</span>
                        <span>{item.timestamp}</span>
                      </div>
                      <span className="text-xs font-mono bg-neutral-900 text-neutral-300 border border-neutral-700 px-2 py-0.5 font-bold">
                        {item.status}
                      </span>
                    </div>

                    <div className="text-sm font-bold text-white font-sans flex items-start gap-2 pt-1">
                      <Sparkles className="h-4 w-4 text-primary shrink-0 mt-0.5" />
                      <span>"{item.prompt}"</span>
                    </div>

                    <div className="font-mono text-xs text-neutral-400 bg-black p-2 border border-neutral-800 overflow-x-auto space-y-0.5">
                      <div className="text-neutral-500">$ {item.command}</div>
                      {item.tools.map((t, i) => (
                        <div key={i} className="text-primary font-semibold">
                          &gt;&gt; [GET] {t}
                        </div>
                      ))}
                    </div>

                    <p className="text-xs sm:text-sm text-neutral-200 leading-relaxed font-sans pt-1">
                      {item.response}
                    </p>
                  </div>
                ))
              )}
            </div>

            {/* Footer del Panel */}
            <div className="p-4 border-t-2 border-neutral-800 bg-black flex items-center justify-between text-xs font-mono text-neutral-500">
              <span>router-core-agent</span>
              <button
                onClick={() => setIsDrawerOpen(false)}
                className="px-3 py-1 bg-neutral-900 hover:bg-neutral-800 border border-neutral-700 text-white cursor-pointer font-bold uppercase"
              >
                Cerrar (Esc)
              </button>
            </div>
          </aside>
        </div>
      )}
    </>
  );
}
