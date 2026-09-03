import React, { useState, useEffect, useCallback } from "react";
import { 
  AlertCircle, 
  RotateCw,
  Layers,
  ChevronDown,
  ChevronUp,
  Copy,
  Check
} from "lucide-react";
import { ConnectionQuality } from "@/components/ConnectionQuality";
import { HelpGuide } from "@/components/HelpGuide";
import { DeviceManager } from "@/components/DeviceManager";
import { CapabilitiesGrid } from "@/components/CapabilitiesGrid";
import { 
  mockDevice, 
  mockStatus, 
  mockClients, 
  mockCapabilities
} from "@/data/mockData";

const ROUTER_API_URL = import.meta.env.VITE_ROUTER_API_URL || "/api/router";

export function Dashboard({ isLive, onWanStatusChange }) {
  const [liveDeviceData, setLiveDeviceData] = useState(null);
  const [liveStatusData, setLiveStatusData] = useState(null);
  const [liveClientsData, setLiveClientsData] = useState(null);
  const [liveCapsData, setLiveCapsData] = useState(null);
  const [liveError, setLiveError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [showAdvancedTelemetry, setShowAdvancedTelemetry] = useState(false);
  const [copiedIp, setCopiedIp] = useState(false);

  const deviceData = isLive && liveDeviceData ? liveDeviceData : mockDevice;
  const statusData = isLive && liveStatusData ? liveStatusData : mockStatus;
  const clientsData = isLive && liveClientsData ? liveClientsData : mockClients;
  const capsData = isLive && liveCapsData ? liveCapsData : mockCapabilities;

  const handleCopyIp = () => {
    const ip = deviceData.managementAddress || "192.168.1.1";
    navigator.clipboard.writeText(ip);
    setCopiedIp(true);
    setTimeout(() => setCopiedIp(false), 2000);
  };

  const fetchLiveData = useCallback(async () => {
    setLoading(true);
    setLiveError(null);

    try {
      const fetchEndpoint = async (path) => {
        try {
          const res = await fetch(`${ROUTER_API_URL}${path}`);
          const json = await res.json().catch(() => null);
          return json;
        } catch {
          return null;
        }
      };

      const [devRes, statRes, cliRes, capRes] = await Promise.all([
        fetchEndpoint("/v0/device"),
        fetchEndpoint("/v0/status"),
        fetchEndpoint("/v0/clients"),
        fetchEndpoint("/v0/capabilities"),
      ]);

      if (devRes && devRes.vendor) setLiveDeviceData(devRes);
      if (statRes) {
        setLiveStatusData(statRes);
        if (onWanStatusChange && statRes.wanStatus) {
          onWanStatusChange(statRes.wanStatus);
        }
      }
      if (cliRes && cliRes.clients) setLiveClientsData(cliRes);
      if (capRes && capRes.capabilities) setLiveCapsData(capRes);

      if (!devRes && !statRes) {
        setLiveError(`No se pudo conectar con router-core serve en ${ROUTER_API_URL}. Mostrando datos locales de respaldo.`);
      }
    } catch {
      setLiveError(`Error al conectar con ${ROUTER_API_URL}. Inicia el servicio: ./bin/router-core serve --mock (o --host 192.168.1.1)`);
    } finally {
      setLoading(false);
    }
  }, [onWanStatusChange]);

  const refreshData = () => {
    if (isLive) {
      fetchLiveData();
    }
  };

  useEffect(() => {
    if (!isLive) return;

    let cancelled = false;
    const run = async () => {
      await Promise.resolve();
      if (!cancelled) {
        fetchLiveData();
      }
    };
    run();
    return () => {
      cancelled = true;
    };
  }, [isLive, fetchLiveData]);

  const isWanConnected = statusData.wanStatus === "connected" || statusData.reachable === "true";
  const clientCount = clientsData.clients?.length || 0;

  return (
    <div className="w-full px-4 sm:px-6 lg:px-8 py-5 space-y-6 bg-black text-white font-sans">
      {/* Live Error Notification */}
      {liveError && (
        <div className="border-2 border-neutral-700 bg-neutral-900 p-3 text-xs text-rose-300 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-2 font-mono">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-rose-400 shrink-0" />
            <span>{liveError}</span>
          </div>
          <code className="bg-black px-2 py-0.5 text-rose-300 border border-neutral-700 text-xs">
            ./bin/router-core serve --host 192.168.1.1
          </code>
        </div>
      )}

      {/* ========================================================================= */}
      {/* 1. SECCIÓN PRIMORDIAL: DATOS DEL ROUTER, IP Y FIRMWARE (BRUTALISTA ELEVADO) */}
      {/* ========================================================================= */}
      <section className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-4 text-white">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
          <div className="space-y-1.5">
            <div className="flex flex-wrap items-center gap-3">
              <h1 className="text-2xl sm:text-3xl lg:text-4xl font-black uppercase tracking-tight text-white font-sans">
                {deviceData.vendor} <span className="text-primary">{deviceData.model}</span>
              </h1>
              <div className="flex items-center gap-1.5 px-2.5 py-0.5 border border-neutral-800 bg-neutral-900 font-mono text-xs">
                <span className={`h-2 w-2 rounded-full ${isWanConnected ? "bg-emerald-500 animate-pulse" : "bg-rose-500"}`} />
                <span className={`font-bold uppercase ${isWanConnected ? "text-emerald-400" : "text-rose-400"}`}>
                  {isWanConnected ? "INTERNET CONECTADO" : "SIN CONEXIÓN"}
                </span>
              </div>
            </div>
          </div>

          <button
            onClick={refreshData}
            disabled={loading}
            className="px-4 py-2 bg-neutral-900 hover:bg-neutral-800 border-2 border-neutral-700 text-xs font-mono font-bold uppercase tracking-wider text-white flex items-center gap-2 transition-colors cursor-pointer self-start sm:self-auto disabled:opacity-50"
          >
            <RotateCw className={`h-3.5 w-3.5 ${loading ? "animate-spin text-primary" : ""}`} />
            <span>{loading ? "Actualizando..." : "Actualizar"}</span>
          </button>
        </div>

        {/* Minimal Horizontal Telemetry Ribbon (Brutalist style with border-l-2) */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4 pt-1 font-mono text-xs">
          {/* IP con 1-click copy */}
          <div className="border-l-2 border-primary pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">IP DE GESTIÓN:</span>
            <button
              onClick={handleCopyIp}
              className="flex items-center gap-1.5 font-bold text-white hover:text-primary text-sm transition-colors cursor-pointer text-left w-full group"
              title="Clic para copiar dirección IP"
            >
              <span>{deviceData.managementAddress || "192.168.1.1"}</span>
              {copiedIp ? (
                <Check className="h-3.5 w-3.5 text-emerald-400 shrink-0" />
              ) : (
                <Copy className="h-3.5 w-3.5 text-neutral-500 group-hover:text-primary shrink-0" />
              )}
              <span className="text-[10px] text-neutral-500 group-hover:text-primary">
                {copiedIp ? "¡COPIADA!" : "[COPIAR]"}
              </span>
            </button>
          </div>

          {/* Dispositivos Activos */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">DISPOSITIVOS:</span>
            <span className="font-bold text-white text-sm block">
              {clientCount} CONECTADOS
            </span>
          </div>

          {/* Seguridad de Red */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">SEGURIDAD:</span>
            <span className="font-bold text-emerald-400 text-sm block">
              PROTEGIDA (WPA2)
            </span>
          </div>

          {/* Tiempo Encendido */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2">
            <span className="text-neutral-500 uppercase text-xs block font-medium">TIEMPO ACTIVO:</span>
            <span className="font-bold text-white text-sm block truncate">
              {statusData.uptime || "5h 33m 20s"}
            </span>
          </div>

          {/* Versión Firmware */}
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1 bg-black/40 py-1.5 pr-2 col-span-1 sm:col-span-2 lg:col-span-1">
            <span className="text-neutral-500 uppercase text-xs block font-medium">FIRMWARE:</span>
            <span className="font-bold text-white text-sm block truncate" title={deviceData.firmwareVersion?.value}>
              {deviceData.firmwareVersion?.value || "3.15.9 Build 140724"}
            </span>
          </div>
        </div>
      </section>

      {/* 2. CALIDAD DE CONEXIÓN */}
      <section>
        <ConnectionQuality />
      </section>

      {/* 3. DISPOSITIVOS CONECTADOS */}
      <section>
        <DeviceManager 
          clients={clientsData.clients} 
          state={clientsData.state}
        />
      </section>

      {/* 4. GUÍA CLARA Y SIMPLE */}
      <section>
        <HelpGuide />
      </section>

      {/* 5. FUNCIONES DEL ROUTER (Colapsable) */}
      <section className="border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm text-white">
        <div 
          onClick={() => setShowAdvancedTelemetry(!showAdvancedTelemetry)}
          className="flex items-center justify-between cursor-pointer group"
        >
          <div className="flex items-center gap-2.5">
            <Layers className="h-4 w-4 text-primary" />
            <h4 className="text-sm sm:text-base font-bold font-mono uppercase text-white group-hover:text-primary transition-colors">
              Funciones del Router
            </h4>
          </div>

          <div className="p-1 border border-neutral-800 bg-neutral-900 group-hover:border-neutral-700 transition-colors">
            {showAdvancedTelemetry ? (
              <ChevronUp className="h-4 w-4 text-white" />
            ) : (
              <ChevronDown className="h-4 w-4 text-neutral-400" />
            )}
          </div>
        </div>

        {showAdvancedTelemetry && (
          <div className="mt-4 pt-4 border-t-2 border-neutral-800">
            <CapabilitiesGrid
              capabilities={capsData.capabilities}
              isLive={isLive}
              routerApiUrl={ROUTER_API_URL}
            />
          </div>
        )}
      </section>
    </div>
  );
}

export default Dashboard;
