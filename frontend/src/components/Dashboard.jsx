import React, { useState, useEffect } from "react";
import { 
  AlertCircle, 
  RotateCw,
  Layers,
  ChevronDown,
  ChevronUp
} from "lucide-react";
import { NetworkComparisonChart } from "@/components/NetworkComparisonChart";
import { GrandpaGuideSection } from "@/components/GrandpaGuideSection";
import { BrutalistClientsManager } from "@/components/BrutalistClientsManager";
import { CapabilitiesGrid } from "@/components/CapabilitiesGrid";
import { 
  mockDevice, 
  mockStatus, 
  mockClients, 
  mockCapabilities
} from "@/data/mockData";

export function Dashboard({ isLive, onWanStatusChange }) {
  const [deviceData, setDeviceData] = useState(mockDevice);
  const [statusData, setStatusData] = useState(mockStatus);
  const [clientsData, setClientsData] = useState(mockClients);
  const [capsData, setCapsData] = useState(mockCapabilities);
  const [liveError, setLiveError] = useState(null);
  const [loading, setLoading] = useState(false);
  const [showAdvancedTelemetry, setShowAdvancedTelemetry] = useState(false);

  const refreshData = async () => {
    if (!isLive) {
      setDeviceData(mockDevice);
      setStatusData(mockStatus);
      setClientsData(mockClients);
      setCapsData(mockCapabilities);
      setLiveError(null);
      if (onWanStatusChange && mockStatus.wanStatus) {
        onWanStatusChange(mockStatus.wanStatus);
      }
      return;
    }

    setLoading(true);
    setLiveError(null);

    try {
      const [devRes, statRes, cliRes, capRes] = await Promise.all([
        fetch("http://127.0.0.1:8484/v0/device").then((r) => (r.ok ? r.json() : null)),
        fetch("http://127.0.0.1:8484/v0/status").then((r) => (r.ok ? r.json() : null)),
        fetch("http://127.0.0.1:8484/v0/clients").then((r) => (r.ok ? r.json() : null)),
        fetch("http://127.0.0.1:8484/v0/capabilities").then((r) => (r.ok ? r.json() : null)),
      ]);

      if (devRes) setDeviceData(devRes);
      if (statRes) {
        setStatusData(statRes);
        if (onWanStatusChange && statRes.wanStatus) {
          onWanStatusChange(statRes.wanStatus);
        }
      }
      if (cliRes) setClientsData(cliRes);
      if (capRes) setCapsData(capRes);

      if (!devRes && !statRes) {
        setLiveError("No se pudo conectar con router-core serve en http://127.0.0.1:8484. Mostrando datos locales de respaldo.");
      }
    } catch {
      setLiveError("Error al conectar con 127.0.0.1:8484. Asegúrate de ejecutar: ./bin/router-core serve --host 192.168.1.1");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshData();
  }, [isLive]);

  const trackerItems = Object.entries(capsData.capabilities || {}).map(([name, status]) => ({
    name,
    status
  }));
  const verifiedCount = trackerItems.filter((i) => i.status === "verified").length;

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
      {/* 1. SECCIÓN PRIMORDIAL: DATOS DEL ROUTER, IP Y FIRMWARE (COMPACTO DARK) */}
      {/* ========================================================================= */}
      <section className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-3 text-white">
        <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
          <div>
            <h1 className="text-2xl sm:text-3xl lg:text-4xl font-black uppercase tracking-tight text-white font-sans">
              {deviceData.vendor} <span className="text-primary">{deviceData.model}</span>
            </h1>
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

        {/* Minimal Horizontal Telemetry */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-6 pt-3 pb-1 font-mono text-xs">
          <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
            <span className="text-neutral-500 uppercase text-xs block font-medium">IP DE GESTIÓN:</span>
            <span className="font-bold text-white text-sm block">{deviceData.managementAddress}</span>
          </div>

          <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
            <span className="text-neutral-500 uppercase text-xs block font-medium">FIRMWARE:</span>
            <span className="font-bold text-white text-sm truncate block" title={deviceData.firmwareVersion?.value}>
              {deviceData.firmwareVersion?.value || "3.15.9 Build 140724"}
            </span>
          </div>

          <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
            <span className="text-neutral-500 uppercase text-xs block font-medium">TIEMPO ENCENDIDO:</span>
            <span className="font-bold text-white text-sm block">{statusData.uptime || "5h 33m 20s"}</span>
          </div>

          <div className="border-l-2 border-neutral-800 pl-3 space-y-1">
            <span className="text-neutral-500 uppercase text-xs block font-medium">ESTADO WAN:</span>
            <div className="flex items-center gap-1.5 pt-0.5">
              <span className={`h-2 w-2 rounded-full ${statusData.wanStatus === "connected" ? "bg-emerald-500 animate-pulse" : "bg-rose-500"}`} />
              <span className={`font-bold text-sm uppercase ${statusData.wanStatus === "connected" ? "text-emerald-400" : "text-rose-400"}`}>
                {statusData.wanStatus || "CONNECTED"}
              </span>
            </div>
          </div>
        </div>
      </section>

      {/* ========================================================================= */}
      {/* 2. GRÁFICO DE RED DE UN ANTES Y DESPUÉS */}
      {/* ========================================================================= */}
      <section>
        <NetworkComparisonChart />
      </section>

      {/* Dispositivos conectados con opción de ponerles alias */}
      <section>
        <BrutalistClientsManager 
          clients={clientsData.clients} 
          state={clientsData.state}
        />
      </section>

      {/* ========================================================================= */}
      {/* 6. MATRIZ DE CAPACIDADES Y AUDITORÍA TÉCNICA (MINIMALISTA Y COLAPSABLE) */}
      {/* ========================================================================= */}
      <section className="border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm text-white">
        <div 
          onClick={() => setShowAdvancedTelemetry(!showAdvancedTelemetry)}
          className="flex items-center justify-between cursor-pointer group"
        >
          <div className="flex items-center gap-2.5">
            <Layers className="h-4 w-4 text-primary" />
            <div>
              <h4 className="text-xs sm:text-sm font-bold font-mono uppercase text-white group-hover:text-primary transition-colors">
                MATRIZ TÉCNICA DE CAPACIDADES
              </h4>
              <p className="text-xs text-neutral-300 font-sans">
                Es la lista de funciones que router-core puede leer del router, como el estado, los aparatos y la seguridad.
              </p>
              <p className="text-xs text-neutral-500 font-sans">
                {verifiedCount} de {trackerItems.length} superficies observadas en hardware real TL-WR841N
              </p>
            </div>
          </div>

          <div className="flex items-center gap-2 font-mono text-xs">
            <span className="text-neutral-300 bg-neutral-900 border border-neutral-700 px-2 py-0.5 font-semibold">
              {showAdvancedTelemetry ? "Ocultar" : "Expandir"}
            </span>
            {showAdvancedTelemetry ? (
              <ChevronUp className="h-4 w-4 text-white" />
            ) : (
              <ChevronDown className="h-4 w-4 text-neutral-500" />
            )}
          </div>
        </div>

        {showAdvancedTelemetry && (
          <div className="mt-4 pt-4 border-t-2 border-neutral-800 space-y-4">
            <CapabilitiesGrid capabilities={capsData.capabilities} />

            <div className="border border-neutral-800 bg-black p-3 font-mono text-xs text-neutral-400 space-y-1">
              <div className="text-white font-bold">
                INVARIANTE DE SEGURIDAD (ADR-0001):
              </div>
              <p>
                router-core nunca ejecuta mutaciones sobre el router. Todas las operaciones son peticiones GET seguras hacia 127.0.0.1 o RFC1918.
              </p>
            </div>
          </div>
        )}
      </section>

      <section>
        <GrandpaGuideSection
          clientCount={clientsData.clients?.length || 3}
          isWanConnected={statusData.wanStatus === "connected" || statusData.reachable === "true"}
        />
      </section>
    </div>
  );
}

export default Dashboard;
