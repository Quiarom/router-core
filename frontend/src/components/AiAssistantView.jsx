import React, { useEffect, useState } from "react";
import {
  AlertCircle,
  CornerDownLeft,
  RotateCw,
  Sparkles,
  Trash2
} from "lucide-react";

const AGENT_API_URL =
  import.meta.env.VITE_AGENT_API_URL || "http://127.0.0.1:8585/v0/chat";
const AGENT_HEALTH_URL = AGENT_API_URL.replace(/\/v0\/chat$/, "/healthz");

const suggestedQuestions = [
  "¿Está expuesta mi red al exterior?",
  "¿Qué aparatos están conectados ahora?",
  "¿Tengo DMZ o reenvío de puertos activados?",
  "¿Qué información falta para evaluar la seguridad del Wi-Fi?"
];

export function AiAssistantView() {
  const [query, setQuery] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [isCheckingConnection, setIsCheckingConnection] = useState(false);
  const [connectionState, setConnectionState] = useState("unknown");
  const [errorMessage, setErrorMessage] = useState("");
  const [conversations, setConversations] = useState([]);

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "auto" });
  }, []);

  const checkConnection = async () => {
    setIsCheckingConnection(true);
    setErrorMessage("");
    try {
      const response = await fetch(AGENT_HEALTH_URL);
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }
      setConnectionState("connected");
    } catch {
      setConnectionState("disconnected");
      setErrorMessage(
        "No se pudo conectar con router-core-agent en 127.0.0.1:8585."
      );
    } finally {
      setIsCheckingConnection(false);
    }
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

      setConnectionState("connected");
      setConversations((current) =>
        current.map((conversation) =>
          conversation.id === pendingId
            ? {
                ...conversation,
                isPending: false,
                answer: payload.answer,
                steps: payload.steps || [],
                model: payload.model,
                mode: payload.mode
              }
            : conversation
        )
      );
    } catch (error) {
      setConnectionState("disconnected");
      setConversations((current) =>
        current.filter((conversation) => conversation.id !== pendingId)
      );
      setErrorMessage(
        error instanceof Error
          ? error.message
          : "No se pudo completar la consulta con MiniMax."
      );
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <div className="w-full max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-6 bg-black text-white font-sans">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
        <div>
          <div className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-primary" />
            <h1 className="text-2xl sm:text-3xl font-black uppercase tracking-tight">
              Asistente de red
            </h1>
          </div>
          <p className="mt-1 text-sm text-neutral-400">
            MiniMax M3 razona sobre observaciones de solo lectura de router-core.
          </p>
        </div>

        <button
          onClick={checkConnection}
          disabled={isCheckingConnection}
          className="px-4 py-2 bg-neutral-900 hover:bg-neutral-800 border-2 border-neutral-700 text-xs font-mono font-bold uppercase text-white flex items-center gap-2 disabled:opacity-50 cursor-pointer"
        >
          <span
            className={`h-2.5 w-2.5 rounded-full ${
              connectionState === "connected"
                ? "bg-emerald-400"
                : connectionState === "disconnected"
                  ? "bg-rose-500"
                  : "bg-neutral-500"
            }`}
          />
          <RotateCw
            className={`h-3.5 w-3.5 ${isCheckingConnection ? "animate-spin" : ""}`}
          />
          Comprobar agente
        </button>
      </div>

      <div className="space-y-3">
        <div className="flex flex-col sm:flex-row gap-2 items-stretch">
          <input
            type="text"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") handleExecute();
            }}
            placeholder="Pregunta algo sobre tu router..."
            autoFocus
            className="w-full h-14 bg-black border-2 border-neutral-800 focus:border-primary focus:outline-none px-5 text-base text-white placeholder-neutral-500"
          />
          <button
            onClick={() => handleExecute()}
            disabled={isProcessing || !query.trim()}
            className="h-14 px-6 bg-primary hover:bg-primary-hover text-white font-bold uppercase tracking-wider text-xs font-mono disabled:opacity-40 disabled:cursor-not-allowed flex items-center justify-center gap-2 cursor-pointer"
          >
            {isProcessing ? (
              <>
                <span className="animate-spin">⟳</span>
                Analizando...
              </>
            ) : (
              <>
                <CornerDownLeft className="h-4 w-4" />
                Consultar
              </>
            )}
          </button>
        </div>

        <div className="flex flex-wrap gap-2">
          {suggestedQuestions.map((question) => (
            <button
              key={question}
              onClick={() => handleExecute(question)}
              disabled={isProcessing}
              className="border border-neutral-800 bg-neutral-950 px-3 py-1.5 text-xs text-neutral-400 hover:border-primary hover:text-white disabled:opacity-40 cursor-pointer"
            >
              {question}
            </button>
          ))}
        </div>
      </div>

      {errorMessage && (
        <div className="border-2 border-rose-900 bg-rose-950/30 p-4 text-sm text-rose-200 space-y-2">
          <div className="flex items-start gap-2">
            <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
            <span>{errorMessage}</span>
          </div>
          <code className="block text-xs text-neutral-300 font-mono">
            OPENROUTER_API_KEY=&lt;TU_CLAVE&gt; ./bin/router-core-agent --serve 127.0.0.1:8585
          </code>
        </div>
      )}

      <div className="flex items-center justify-between border-b border-neutral-800 pb-3">
        <span className="text-xs font-mono uppercase text-neutral-500">
          {conversations.length === 0
            ? "Escribe una pregunta para comenzar"
            : `${conversations.length} consultas`}
        </span>
        {conversations.length > 0 && (
          <button
            onClick={() => setConversations([])}
            className="text-neutral-500 hover:text-rose-400 flex items-center gap-1.5 text-xs font-mono uppercase cursor-pointer"
          >
            <Trash2 className="h-3.5 w-3.5" />
            Limpiar
          </button>
        )}
      </div>

      <div className="space-y-8">
        {conversations.map((conversation) => (
          <article key={conversation.id} className="space-y-4 border-b border-neutral-900 pb-8">
            <div className="flex justify-end">
              <div className="max-w-2xl border border-neutral-800 bg-neutral-900 px-4 py-2.5">
                <p className="text-sm sm:text-base text-white">{conversation.question}</p>
                <span className="block mt-1 text-right text-xs text-neutral-500 font-mono">
                  {conversation.timestamp}
                </span>
              </div>
            </div>

            {conversation.isPending ? (
              <div className="flex items-center gap-2 text-primary font-mono text-xs font-bold uppercase">
                <span className="animate-spin">⟳</span>
                Consultando observaciones y esperando a MiniMax M3...
              </div>
            ) : (
              <div className="max-w-4xl space-y-4">
                <div className="flex flex-wrap items-center gap-2 text-xs font-mono">
                  <span className="border border-neutral-800 bg-neutral-950 px-2 py-1 text-neutral-300">
                    {conversation.model}
                  </span>
                  <span
                    className={`border px-2 py-1 ${
                      conversation.mode === "live"
                        ? "border-emerald-900 text-emerald-400"
                        : "border-amber-900 text-amber-400"
                    }`}
                  >
                    {conversation.mode === "live" ? "MINIMAX EN VIVO" : "MODO DEMOSTRACIÓN"}
                  </span>
                </div>

                <div className="border border-neutral-800 bg-neutral-950 p-3 space-y-2 overflow-x-auto">
                  <div className="text-xs font-mono uppercase text-neutral-500">
                    Observaciones consultadas
                  </div>
                  {conversation.steps.map((step, index) => (
                    <div key={`${step.path}-${index}`} className="font-mono text-xs">
                      <div className="text-primary">
                        &gt;&gt; [GET {step.http_status}] {step.path}
                      </div>
                      <pre className="mt-1 whitespace-pre-wrap break-all text-neutral-500">
                        {JSON.stringify(step.result)}
                      </pre>
                    </div>
                  ))}
                </div>

                <p className="whitespace-pre-wrap text-sm sm:text-base text-neutral-200 leading-relaxed">
                  {conversation.answer}
                </p>
              </div>
            )}
          </article>
        ))}
      </div>
    </div>
  );
}

export default AiAssistantView;
