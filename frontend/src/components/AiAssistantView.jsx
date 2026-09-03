import React, { useEffect, useState } from "react";
import {
  AlertCircle,
  CornerDownLeft,
  RotateCw,
  Trash2,
  ChevronDown,
  ChevronUp,
  Bot
} from "lucide-react";
import { mockDevice, mockStatus, mockClients, mockCapabilities } from "@/data/mockData";

const AGENT_API_URL =
  import.meta.env.VITE_AGENT_API_URL || "http://127.0.0.1:8585/v0/chat";

export function AiAssistantView({ isLive = false }) {
  const [query, setQuery] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [errorMessage, setErrorMessage] = useState("");
  const [conversations, setConversations] = useState([]);
  const [expandedObservations, setExpandedObservations] = useState({});

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "auto" });
  }, []);

  const toggleObservation = (id) => {
    setExpandedObservations((prev) => ({
      ...prev,
      [id]: !prev[id]
    }));
  };

  const handleExecute = async (customPrompt) => {
    const question = (customPrompt || query).trim();
    if (!question || isProcessing) return;

    const pendingId = `turn-${Date.now()}`;
    setIsProcessing(true);
    setErrorMessage("");
    setQuery("");
    setConversations((current) => [
      ...current,
      {
        id: pendingId,
        question,
        timestamp: new Date().toLocaleTimeString("es-AR", {
          hour: "2-digit",
          minute: "2-digit"
        }),
        isPending: true,
        answer: "",
        steps: []
      }
    ]);

    if (!isLive) {
      setTimeout(() => {
        const mockSteps = [
          { tool: "get_device", path: "/v0/device", http_status: 200, result: mockDevice },
          { tool: "get_status", path: "/v0/status", http_status: 200, result: mockStatus },
          { tool: "get_clients", path: "/v0/clients", http_status: 200, result: mockClients },
          { tool: "get_capabilities", path: "/v0/capabilities", http_status: 200, result: mockCapabilities }
        ];

        let mockAnswer = "Hechos observados en simulación local:\n\n" +
          "• **Dispositivo**: " + mockDevice.vendor + " " + mockDevice.model + " (Firmware " + mockDevice.firmwareVersion.value + ").\n" +
          "• **Estado WAN**: " + (mockStatus.wanStatus === "connected" ? "Conectado a Internet" : "Sin conexión") + " con " + (mockStatus.uptime || "5h 33m") + " de actividad.\n" +
          "• **Equipos observados**: " + mockClients.clients.length + " dispositivos con concesión DHCP activa.\n" +
          "• **Capacidades verificadas**: Dispositivo, estado, clientes y DMZ.\n\n" +
          "Para auditoría en vivo con el modelo MiniMax M3 y conexión directa al router, cambia a modo 'Live' en la barra superior.";

        const qLower = question.toLowerCase();
        if (qLower.includes("quien") || qLower.includes("quién") || qLower.includes("dispositiv") || qLower.includes("conectad") || qLower.includes("aparato")) {
          mockAnswer = "Equipos observados en la red según la tabla DHCP local:\n\n" +
            mockClients.clients.map(c => `• **${c.name}** (IP: \`${c.ip}\`, MAC: \`${c.mac}\`, Concesión: ${c.lease})`).join("\n") +
            "\n\nLímites de la evidencia: Se observan " + mockClients.clients.length + " equipos con concesión DHCP activa.";
        } else if (qLower.includes("wifi") || qLower.includes("wi-fi") || qLower.includes("expuest") || qLower.includes("segur")) {
          mockSteps.push({
            tool: "get_security",
            path: "/v0/security/wireless",
            http_status: 503,
            result: { state: "unavailable", reason: "router-core: endpoint is unverified against captured traffic" }
          });
          mockAnswer = "Análisis de exposición de red Wi-Fi (Modo Mock):\n\n" +
            "• **Seguridad Wi-Fi**: El endpoint de seguridad inalámbrica está en estado `unavailable` (pendiente de verificación contra capturas de tráfico).\n" +
            "• **WPS**: Está en estado `absent` (HTTP 404), no hay superficie vulnerable de WPS en este firmware.\n" +
            "• **DMZ**: Está deshabilitado (`false`), ningún equipo interno está expuesto completamente a la WAN.\n\n" +
            "Recomendación: En modo Live con MiniMax M3 activo, el agente audita estas superficies paso a paso.";
        }

        setConversations((current) =>
          current.map((conversation) =>
            conversation.id === pendingId
              ? {
                  ...conversation,
                  isPending: false,
                  answer: mockAnswer,
                  model: "MiniMax M3 (Simulación Mock)",
                  mode: "mock",
                  steps: mockSteps
                }
              : conversation
          )
        );
        setIsProcessing(false);
      }, 500);
      return;
    }

    try {
      const response = await fetch(AGENT_API_URL, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question })
      });
      const payload = await response.json().catch(() => ({}));
      if (!response.ok) {
        throw new Error(payload.error || `El agente respondió HTTP ${response.status}`);
      }

      setConversations((current) =>
        current.map((conversation) =>
          conversation.id === pendingId
            ? {
                ...conversation,
                isPending: false,
                answer:
                  payload.answer ||
                  "No se recibió respuesta explicativa del agente.",
                model: payload.model || "MiniMax M3",
                mode: payload.mode || "live",
                steps: payload.steps || []
              }
            : conversation
        )
      );
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "Error desconocido al consultar el agente."
      );
      setConversations((current) =>
        current.filter((conversation) => conversation.id !== pendingId)
      );
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <div className="w-full max-w-5xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6 text-white font-sans">
      {/* Header section: Brutalist & Tech-forward */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-4">
        <h1 className="text-xl sm:text-2xl font-black uppercase tracking-tight font-mono text-white flex items-center gap-2">
          <span className="text-primary font-mono font-black">&gt;&gt;</span> ASISTENTE DE RED (MINIMAX M3)
        </h1>
        <div className="flex items-center gap-2 font-mono text-xs">
          <span className={`px-2.5 py-1 border font-bold uppercase ${
            isLive
              ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-400"
              : "border-amber-500/40 bg-amber-500/10 text-amber-400"
          }`}>
            {isLive ? "MODO LIVE (CONEXIÓN ACTIVA)" : "MODO MOCK (FIXTURES)"}
          </span>
        </div>
      </div>

      {/* Input Box: Brutalist */}
      <div className="space-y-3">
        <div className="border-2 border-neutral-800 bg-neutral-950 p-2 flex flex-col sm:flex-row gap-2 items-center focus-within:border-primary transition-colors">
          <input
            type="text"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") handleExecute();
            }}
            placeholder="Escribe tu consulta sobre el router o los dispositivos..."
            autoFocus
            className="w-full h-11 bg-transparent px-3 text-sm text-white placeholder-neutral-500 font-sans focus:outline-none"
          />
          <button
            onClick={() => handleExecute()}
            disabled={isProcessing || !query.trim()}
            className="w-full sm:w-auto h-10 px-5 bg-primary hover:bg-primary-hover text-white font-mono font-bold uppercase text-xs sm:text-sm flex items-center justify-center gap-1.5 transition-colors disabled:opacity-40 disabled:cursor-not-allowed shrink-0 cursor-pointer"
          >
            {isProcessing ? (
              <>
                <RotateCw className="h-3.5 w-3.5 animate-spin" />
                <span>Analizando...</span>
              </>
            ) : (
              <>
                <CornerDownLeft className="h-3.5 w-3.5" />
                <span>Preguntar</span>
              </>
            )}
          </button>
        </div>
      </div>

      {/* Error alert: Brutalist red banner */}
      {errorMessage && (
        <div className="border-2 border-rose-600 bg-neutral-950 p-3.5 text-xs text-rose-300 space-y-1.5 font-mono">
          <div className="flex items-center gap-2 font-bold uppercase">
            <AlertCircle className="h-4 w-4 text-rose-400 shrink-0" />
            <span>{errorMessage}</span>
          </div>
          <p className="text-neutral-400 text-xs pl-6">
            Para iniciar el servicio con MiniMax: <code className="text-white bg-neutral-900 px-1.5 py-0.5 border border-neutral-800">./bin/router-core-agent --serve 127.0.0.1:8585</code>
          </p>
        </div>
      )}

      {/* Conversation history header (only shown when conversations exist) */}
      {conversations.length > 0 && (
        <div className="flex items-center justify-between pt-2 text-xs font-mono text-neutral-400">
          <span>
            {`${conversations.length} ${conversations.length === 1 ? "CONSULTA" : "CONSULTAS"}`}
          </span>
          <button
            onClick={() => setConversations([])}
            className="hover:text-rose-400 flex items-center gap-1 transition-colors cursor-pointer"
          >
            <Trash2 className="h-3 w-3" />
            <span>LIMPIAR HISTORIAL</span>
          </button>
        </div>
      )}

      {/* Chat Messages */}
      <div className="space-y-6">
        {conversations.map((conversation) => (
          <article key={conversation.id} className="space-y-3 font-sans">
            {/* User message */}
            <div className="flex justify-end">
              <div className="max-w-xl border-2 border-neutral-700 bg-neutral-900 px-4 py-2.5 text-sm text-white shadow-sm">
                <p>{conversation.question}</p>
                <span className="block mt-1 text-right text-[11px] text-neutral-400 font-mono">
                  {conversation.timestamp}
                </span>
              </div>
            </div>

            {/* Assistant response */}
            {conversation.isPending ? (
              <div className="flex items-center gap-2.5 text-primary text-xs font-mono font-bold py-2 px-1">
                <RotateCw className="h-3.5 w-3.5 animate-spin" />
                <span>CONSULTANDO ROUTER Y RAZONANDO CON MINIMAX...</span>
              </div>
            ) : (
              <div className="border-2 border-neutral-800 bg-neutral-950 p-5 space-y-4 shadow-sm">
                {/* Meta header */}
                <div className="flex flex-wrap items-center justify-between gap-2 border-b border-neutral-800 pb-3">
                  <div className="flex items-center gap-2 font-mono">
                    <div className="h-5 w-5 bg-primary/20 border border-primary flex items-center justify-center text-primary">
                      <Bot className="h-3 w-3" />
                    </div>
                    <span className="text-xs font-bold text-white uppercase">
                      {conversation.model || "MiniMax M3"}
                    </span>
                    <span
                      className={`text-[10px] px-2 py-0.5 border font-bold uppercase ${
                        conversation.mode === "live"
                          ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/30"
                          : "bg-amber-500/10 text-amber-400 border-amber-500/30"
                      }`}
                    >
                      {conversation.mode === "live" ? "EN VIVO" : "DEMOSTRACIÓN"}
                    </span>
                  </div>
                </div>

                {/* Main answer text */}
                <div className="text-sm text-neutral-200 leading-relaxed whitespace-pre-wrap font-sans">
                  {conversation.answer}
                </div>

                {/* Collapsible technical observations */}
                {conversation.steps && conversation.steps.length > 0 && (
                  <div className="border-t border-neutral-800 pt-3">
                    <button
                      onClick={() => toggleObservation(conversation.id)}
                      className="flex items-center gap-1.5 text-xs font-mono font-bold text-neutral-400 hover:text-white uppercase transition-colors cursor-pointer"
                    >
                      <span>DATOS TÉCNICOS OBSERVADOS ({conversation.steps.length})</span>
                      {expandedObservations[conversation.id] ? (
                        <ChevronUp className="h-3.5 w-3.5 text-primary" />
                      ) : (
                        <ChevronDown className="h-3.5 w-3.5" />
                      )}
                    </button>

                    {expandedObservations[conversation.id] && (
                      <div className="mt-2.5 border border-neutral-800 bg-black p-3 space-y-2 text-xs font-mono">
                        {conversation.steps.map((step, index) => (
                          <div key={`${step.path}-${index}`} className="space-y-1">
                            <div className="text-primary flex items-center gap-1.5 font-bold">
                              <span className="text-neutral-400">GET {step.http_status}</span>
                              <span>{step.path}</span>
                            </div>
                            <pre className="text-neutral-400 text-[11px] whitespace-pre-wrap break-all overflow-x-auto">
                              {JSON.stringify(step.result, null, 2)}
                            </pre>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )}
          </article>
        ))}
      </div>
    </div>
  );
}

export default AiAssistantView;
