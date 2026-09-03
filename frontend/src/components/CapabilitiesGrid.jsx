import React, { useState, useEffect } from "react";
import { X, RotateCw } from "lucide-react";
import { mockSecurity } from "@/data/mockData";
import { getRouterData } from "@/lib/desktop";

const getCapabilityMeta = (key) => {
  switch (key) {
    case "device":
      return {
        label: "Identidad del Dispositivo",
        path: "/v0/device",
        desc: "Fabricante, modelo y versión de firmware auditada."
      };
    case "status":
      return {
        label: "Estado del Sistema y WAN",
        path: "/v0/status",
        desc: "Enlace exterior activo, verificación de alcance y tiempo de actividad."
      };
    case "clients":
      return {
        label: "Clientes DHCP",
        path: "/v0/clients",
        desc: "Equipos activos con asignación de IP y MAC en la red local."
      };
    case "wireless_security":
      return {
        label: "Seguridad Inalámbrica (Wi-Fi)",
        path: "/v0/security/wireless",
        desc: mockSecurity.wireless.description
      };
    case "dmz":
      return {
        label: "Zona Desmilitarizada (DMZ)",
        path: "/v0/security/dmz",
        desc: mockSecurity.dmz.description
      };
    case "forwarding":
      return {
        label: "Reenvío de Puertos",
        path: "/v0/security/forwarding",
        desc: mockSecurity.forwarding.description
      };
    case "wps":
      return {
        label: "WPS (Wi-Fi Protected Setup)",
        path: "/v0/security/wps",
        desc: mockSecurity.wps.description
      };
    case "upnp":
      return {
        label: "UPnP (Plug & Play)",
        path: "/v0/security/upnp",
        desc: mockSecurity.upnp.description
      };
    case "remote_management":
      return {
        label: "Gestión Remota",
        path: "/v0/security/remote-management",
        desc: mockSecurity.remote_management.description
      };
    default:
      return {
        label: key,
        path: `/v0/${key}`,
        desc: "Capacidad estándar de observación."
      };
  }
};

const getStatusBadge = (state) => {
  switch (state) {
    case "verified":
      return {
        text: "VERIFICADO",
        className: "border border-emerald-500/40 bg-emerald-500/10 text-emerald-400 font-mono text-[11px] font-bold uppercase"
      };
    case "absent":
      return {
        text: "AUSENTE (404)",
        className: "border border-neutral-800 bg-neutral-900 text-neutral-500 font-mono text-[11px] font-bold uppercase"
      };
    case "unsupported_or_unverified":
    case "unverified":
      return {
        text: "POR VERIFICAR",
        className: "border border-amber-500/40 bg-amber-500/10 text-amber-400 font-mono text-[11px] font-bold uppercase"
      };
    case "unavailable":
      return {
        text: "NO DISPONIBLE",
        className: "border border-rose-500/40 bg-rose-500/10 text-rose-400 font-mono text-[11px] font-bold uppercase"
      };
    default:
      return {
        text: (state || "").toUpperCase(),
        className: "border border-neutral-800 bg-neutral-900 text-neutral-400 font-mono text-[11px] font-bold uppercase"
      };
  }
};

export function CapabilitiesGrid({ capabilities = {}, isLive = false }) {
  const [selectedCap, setSelectedCap] = useState(null);
  const [liveResponse, setLiveResponse] = useState(null);
  const [loadingLive, setLoadingLive] = useState(false);

  useEffect(() => {
    if (!selectedCap || !isLive) return;
    const meta = getCapabilityMeta(selectedCap);
    let cancelled = false;
    getRouterData(meta.path)
      .then((response) => {
        if (!cancelled) setLiveResponse(response);
      })
      .catch((error) => {
        if (!cancelled) setLiveResponse({ status: 0, error: String(error) });
      })
      .finally(() => {
        if (!cancelled) setLoadingLive(false);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedCap, isLive]);

  const handleSelectCap = (key) => {
    setLiveResponse(null);
    setLoadingLive(isLive);
    setSelectedCap(key);
  };

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 font-sans">
        {Object.entries(capabilities).map(([key, state]) => {
          const meta = getCapabilityMeta(key);
          const badge = getStatusBadge(state);

          return (
            <div
              key={key}
              onClick={() => handleSelectCap(key)}
              className="cursor-pointer border-2 border-neutral-800 bg-neutral-900/60 hover:border-neutral-600 p-4 sm:p-5 transition-colors space-y-3 group"
            >
              {/* Title & Badge (Without icon, enlarged text) */}
              <div className="flex items-start justify-between gap-3">
                <h5 className="font-bold text-sm sm:text-base text-white group-hover:text-primary transition-colors tracking-tight font-mono">
                  {meta.label}
                </h5>
                <span className={`px-2 py-0.5 shrink-0 ${badge.className}`}>
                  {badge.text}
                </span>
              </div>

              {/* Description (Enlarged) */}
              <p className="text-xs sm:text-sm text-neutral-300 line-clamp-2 leading-relaxed font-sans">
                {meta.desc}
              </p>

              {/* Path and Details Link (Enlarged) */}
              <div className="flex items-center justify-between text-xs font-mono text-neutral-400 border-t border-neutral-800/80 pt-2.5">
                <span className="text-xs font-bold text-neutral-400">{meta.path}</span>
                <span className="text-primary text-xs font-bold group-hover:underline">VER DETALLE &rarr;</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Modal detail */}
      {selectedCap && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-xs">
          <div className="w-full max-w-lg border-2 border-neutral-800 bg-neutral-950 p-6 shadow-2xl space-y-4 text-white font-mono">
            <div className="flex items-center justify-between pb-3 border-b-2 border-neutral-800">
              <h3 className="font-bold uppercase tracking-tight text-base text-white">
                {getCapabilityMeta(selectedCap).label}
              </h3>
              <button
                onClick={() => setSelectedCap(null)}
                className="text-neutral-400 hover:text-white p-1 cursor-pointer transition-colors"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <span className="text-neutral-400 block mb-1 text-[11px] uppercase">ENDPOINT LOCAL:</span>
                <code className="bg-black border border-neutral-800 px-2.5 py-1.5 font-mono text-primary block text-sm">
                  GET {getCapabilityMeta(selectedCap).path}
                </code>
              </div>

              <div>
                <span className="text-neutral-400 block mb-1 text-[11px] uppercase">DESCRIPCIÓN:</span>
                <p className="text-neutral-300 font-sans leading-relaxed text-sm">{getCapabilityMeta(selectedCap).desc}</p>
              </div>

              <div>
                <span className="text-neutral-400 block mb-1 text-[11px] uppercase">ESTADO EN EL CONTRATO:</span>
                <div className="bg-black border border-neutral-800 p-3 text-xs text-neutral-300 font-sans">
                  {capabilities[selectedCap] === "verified" && "✓ Verificado contra hardware real compatible."}
                  {capabilities[selectedCap] === "absent" && "ℹ Característica ausente en este firmware (HTTP 404)."}
                  {capabilities[selectedCap] === "unsupported_or_unverified" && "⏳ Capacidad reservada aún no observada en capturas."}
                  {capabilities[selectedCap] === "unavailable" && "⚠ Servicio no disponible temporalmente (HTTP 503)."}
                </div>
              </div>

              {isLive && (
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-neutral-400 text-[11px] uppercase">RESPUESTA EN VIVO DEL ROUTER:</span>
                    {loadingLive && (
                      <span className="flex items-center gap-1 text-[10px] text-primary">
                        <RotateCw className="h-2.5 w-2.5 animate-spin" />
                        <span>Consultando...</span>
                      </span>
                    )}
                    {liveResponse && (
                      <span className={`text-[10px] px-1.5 py-0.5 border font-bold uppercase ${
                        liveResponse.status === 200
                          ? "border-emerald-500/40 bg-emerald-500/10 text-emerald-400"
                          : liveResponse.status === 404
                          ? "border-neutral-700 bg-neutral-900 text-neutral-400"
                          : "border-rose-500/40 bg-rose-500/10 text-rose-400"
                      }`}>
                        HTTP {liveResponse.status || "ERR"}
                      </span>
                    )}
                  </div>
                  <pre className="bg-black border border-neutral-800 p-3 text-xs text-neutral-300 font-mono overflow-x-auto max-h-40 leading-tight">
                    {loadingLive
                      ? "Consultando API en vivo..."
                      : liveResponse?.data
                      ? JSON.stringify(liveResponse.data, null, 2)
                      : liveResponse?.error
                      ? `Error: ${liveResponse.error}`
                      : "Sin respuesta de la API"}
                  </pre>
                </div>
              )}
            </div>

            <div className="pt-2 flex justify-end">
              <button
                onClick={() => setSelectedCap(null)}
                className="px-4 py-2 border-2 border-neutral-800 bg-neutral-900 hover:bg-neutral-800 text-xs font-mono font-bold uppercase text-white transition-colors cursor-pointer"
              >
                CERRAR
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default CapabilitiesGrid;
