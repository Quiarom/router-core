import React, { useState } from "react";
import { Sparkles, Terminal, Cpu } from "lucide-react";
import { Button } from "@/components/ui/button";

export function AgentPlayground() {
  const [question, setQuestion] = useState("¿Está expuesta mi red al exterior?");
  const [isEvaluating, setIsEvaluating] = useState(false);
  const [result, setResult] = useState(null);

  const sampleQuestions = [
    "¿Está expuesta mi red al exterior?",
    "¿Quién está conectado por Wi-Fi o cable?",
    "¿Tengo DMZ o reenvío de puertos activados?",
    "¿Qué versión de firmware tiene el router y es segura?"
  ];

  const handleQuery = (queryText) => {
    const q = queryText || question;
    setQuestion(q);
    setIsEvaluating(true);
    setResult(null);

    // Simulate the reasoning trace of router-core-agent (MiniMax M3 tool sequence)
    setTimeout(() => {
      let tools = [];
      let answer = "";

      if (q.includes("expuesta") || q.includes("puertos") || q.includes("DMZ")) {
        tools = [
          { tool: 'get_security("dmz")', result: '{ "state": "verified", "dmz_enabled": "false", "dmz_host": "" }' },
          { tool: 'get_security("forwarding")', result: '{ "state": "verified", "forwarding_rules": [] }' },
          { tool: 'get_security("remote-management")', result: '{ "state": "unsupported_or_unverified", "http_status": 404 }' }
        ];
        answer = `**Evaluación de Seguridad (MiniMax M3):**\n\n- **DMZ:** Desactivado (\`dmz_enabled: false\`). Ningún host interno está expuesto completamente a internet.\n- **Port Forwarding:** Sin reglas activas. No hay puertos internos abiertos a la WAN pública.\n- **Gestión Remota:** Ausente o deshabilitada en este firmware (HTTP 404).\n\n**Conclusión:** Tu red **NO** está expuesta al exterior a través de la WAN. Tu configuración actual es conservadora y segura.`;
      } else if (q.includes("conectado") || q.includes("Quién")) {
        tools = [
          { tool: 'get_clients()', result: '{ "state": "verified", "clients": [{"name": "omarchy-laptop", "ip": "192.168.1.100"}, {"name": "pixel-phone", "ip": "192.168.1.101"}, {"name": "smart-tv-livingroom", "ip": "192.168.1.105"}] }' }
        ];
        answer = `**Dispositivos detectados en la red:**\n\nSe observaron **3 dispositivos** con concesión DHCP activa:\n1. \`omarchy-laptop\` (192.168.1.100)\n2. \`pixel-phone\` (192.168.1.101)\n3. \`smart-tv-livingroom\` (192.168.1.105)\n\nTodos los dispositivos operan dentro del rango RFC1918 estándar y tienen leases válidos.`;
      } else {
        tools = [
          { tool: 'get_device()', result: '{ "vendor": "TP-Link", "model": "TL-WR841N/ND", "firmwareVersion": "3.15.9 Build 140724" }' },
          { tool: 'get_status()', result: '{ "reachable": "true", "wanStatus": "connected" }' }
        ];
        answer = `**Identidad y Estado del Dispositivo:**\n\n- Router: **TP-Link TL-WR841N/ND v8.4**\n- Firmware observado: \`3.15.9 Build 140724 Rel.63227n\`\n- Estado WAN: Conectado y operativo.\n- Nota de seguridad: El firmware es de 2013-2014. router-core supervisa de forma segura este router con acceso estrictamente de solo lectura.`;
      }

      setResult({ tools, answer });
      setIsEvaluating(false);
    }, 900);
  };

  return (
    <div className="space-y-6">
      {/* Input box */}
      <div className="rounded-xl border border-slate-800 bg-slate-900/40 p-5">
        <div className="flex items-center gap-2 text-xs font-mono text-purple-400 mb-3">
          <Cpu className="h-4 w-4" />
          <span>MINIMAX M3 REASONING INTERFACE</span>
        </div>

        <div className="flex flex-col sm:flex-row gap-3">
          <input
            type="text"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleQuery()}
            placeholder="Escribe una pregunta en lenguaje natural sobre tu router..."
            className="flex-1 rounded-lg border border-slate-800 bg-slate-950 px-4 py-2.5 text-sm text-white placeholder-slate-500 focus:border-emerald-500 focus:outline-none"
          />
          <Button
            onClick={() => handleQuery()}
            disabled={isEvaluating}
            className="gap-2 bg-purple-600 hover:bg-purple-500 text-white"
          >
            {isEvaluating ? (
              <span className="animate-spin">⟳</span>
            ) : (
              <Sparkles className="h-4 w-4" />
            )}
            Consultar Agente
          </Button>
        </div>

        {/* Suggestion pills */}
        <div className="mt-4 flex flex-wrap items-center gap-2 text-xs text-slate-400">
          <span className="text-[11px] text-slate-500 font-mono">Preguntas sugeridas:</span>
          {sampleQuestions.map((sq, i) => (
            <button
              key={i}
              onClick={() => handleQuery(sq)}
              className="rounded-md border border-slate-800 bg-slate-950/60 px-2.5 py-1 text-slate-300 hover:border-slate-700 hover:text-white transition-colors"
            >
              {sq}
            </button>
          ))}
        </div>
      </div>

      {/* Execution Trace & Answer */}
      {result && (
        <div className="space-y-4">
          {/* Trace view */}
          <div className="rounded-xl border border-slate-800 bg-slate-950 p-4">
            <div className="flex items-center justify-between pb-3 border-b border-slate-850">
              <span className="text-xs font-mono text-slate-400 flex items-center gap-2">
                <Terminal className="h-3.5 w-3.5 text-emerald-400" />
                Read-Only Tool-Call Trace (Execution Log)
              </span>
              <span className="text-[11px] font-mono text-emerald-400">0 writes • 100% GET</span>
            </div>

            <div className="mt-3 space-y-2 font-mono text-xs">
              {result.tools.map((t, idx) => (
                <div key={idx} className="rounded bg-slate-900/80 p-2.5 border border-slate-850">
                  <div className="text-purple-300 font-semibold">
                    &rarr; {t.tool}
                  </div>
                  <div className="text-slate-400 text-[11px] mt-1 overflow-x-auto">
                    {t.result}
                  </div>
                </div>
              ))}
            </div>
          </div>

          {/* Model Synthesis */}
          <div className="rounded-xl border border-purple-500/30 bg-purple-950/20 p-5">
            <div className="flex items-center gap-2 text-xs font-mono text-purple-300 mb-2">
              <Sparkles className="h-4 w-4" />
              <span>SÍNTESIS DEL MODELO (MINIMAX M3)</span>
            </div>
            <div className="text-sm text-slate-200 leading-relaxed whitespace-pre-line">
              {result.answer}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
