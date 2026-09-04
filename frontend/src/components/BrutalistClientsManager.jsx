import React, { useState, useEffect } from "react";
import { 
  Smartphone, 
  Tv, 
  Laptop, 
  Monitor, 
  Edit3, 
  Check, 
  X, 
  Search,
  RotateCcw,
  ChevronLeft,
  ChevronRight,
  HelpCircle
} from "lucide-react";

const DEVICES_PER_PAGE = 6;

export function BrutalistClientsManager({ clients = [], state: _state = "verified" }) {
  const [aliases, setAliases] = useState(() => {
    try {
      const saved = localStorage.getItem("router_core_device_aliases");
      if (saved) return JSON.parse(saved);
    } catch (e) {
      console.error(e);
    }
    return {
      "00:11:22:33:44:55": "Computadora Portátil",
      "AA:BB:CC:11:22:33": "El celular del abuelo",
      "10:20:30:40:50:60": "La tele grande del living"
    };
  });

  const [editingMac, setEditingMac] = useState(null);
  const [tempAlias, setTempAlias] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [activeFilter, setActiveFilter] = useState("all");
  const [currentPage, setCurrentPage] = useState(1);

  useEffect(() => {
    try {
      localStorage.setItem("router_core_device_aliases", JSON.stringify(aliases));
    } catch (e) {
      console.error(e);
    }
  }, [aliases]);

  const handleStartEdit = (mac, currentAlias) => {
    setEditingMac(mac);
    setTempAlias(currentAlias);
  };

  const handleSaveAlias = (mac) => {
    if (tempAlias.trim()) {
      setAliases((prev) => ({
        ...prev,
        [mac]: tempAlias.trim()
      }));
    }
    setEditingMac(null);
  };

  const handleResetAlias = (mac) => {
    setAliases((prev) => {
      const updated = { ...prev };
      delete updated[mac];
      return updated;
    });
    setEditingMac(null);
  };

  const getDeviceDetails = (name = "") => {
    const lower = name.toLowerCase();
    if (lower.includes("phone") || lower.includes("pixel") || lower.includes("iphone") || lower.includes("samsung")) {
      return {
        type: "Teléfono Móvil",
        icon: Smartphone,
        connType: "Wi-Fi (Señal 95%)",
        isWifi: true
      };
    }
    if (lower.includes("tv") || lower.includes("smart") || lower.includes("tele") || lower.includes("roku")) {
      return {
        type: "Televisión Smart",
        icon: Tv,
        connType: "Cable de red o Wi-Fi",
        isWifi: false
      };
    }
    if (lower.includes("laptop") || lower.includes("macbook") || lower.includes("notebook") || lower.includes("portatil")) {
      return {
        type: "Computadora Portátil",
        icon: Laptop,
        connType: "Wi-Fi (Señal 90%)",
        isWifi: true
      };
    }
    return {
      type: "Dispositivo",
      icon: Monitor,
      connType: "Wi-Fi / Red",
      isWifi: true
    };
  };

  const filteredClients = clients.filter((client) => {
    const alias = (aliases[client.mac] || client.name || "").toLowerCase();
    const name = (client.name || "").toLowerCase();
    const ip = (client.ip || "").toLowerCase();
    const mac = (client.mac || "").toLowerCase();
    const q = searchQuery.toLowerCase();

    const matchesSearch = alias.includes(q) || name.includes(q) || ip.includes(q) || mac.includes(q);
    if (!matchesSearch) return false;

    if (activeFilter === "wifi") {
      return getDeviceDetails(client.name).isWifi;
    }
    if (activeFilter === "cable") {
      return !getDeviceDetails(client.name).isWifi;
    }
    return true;
  });

  const totalPages = Math.max(1, Math.ceil(filteredClients.length / DEVICES_PER_PAGE));
  const visiblePage = Math.min(currentPage, totalPages);
  const firstVisibleClient = (visiblePage - 1) * DEVICES_PER_PAGE;
  const visibleClients = filteredClients.slice(
    firstVisibleClient,
    firstVisibleClient + DEVICES_PER_PAGE
  );

  return (
    <div className="w-full border-2 border-neutral-800 bg-neutral-950 p-4 sm:p-5 shadow-sm space-y-5 text-white">
      {/* Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b-2 border-neutral-800 pb-5 pt-1">
        <div>
          <h2 className="text-xl sm:text-2xl font-black uppercase tracking-tight text-white font-sans">
            Aparatos en tu Casa
          </h2>
        </div>

        <span className="text-xs font-mono text-neutral-400 bg-neutral-900 border border-neutral-800 px-3 py-1 font-bold uppercase self-start sm:self-auto">
          {clients.length} DETECTADOS
        </span>
      </div>

      {/* Minimal Filter & Search Bar */}
      <div className="flex flex-col gap-3 border-b border-neutral-800 pb-4">
        <div className="flex flex-col sm:flex-row gap-3 items-center justify-between">
          <div className="relative w-full sm:max-w-lg sm:flex-1">
            <Search className="h-5 w-5 text-neutral-500 absolute left-3 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                setCurrentPage(1);
              }}
              placeholder="Buscar por alias, IP o MAC..."
              className="w-full bg-black border-2 border-neutral-800 pl-10 pr-3 py-3 text-sm text-white placeholder-neutral-500 font-sans focus:border-primary focus:outline-none"
            />
          </div>

          <div className="flex items-center gap-1 font-mono text-xs w-full sm:w-auto justify-end">
            <span className="text-neutral-500 mr-2 text-xs">Filtrar:</span>
            {[
              { id: "all", label: "TODOS" },
              { id: "wifi", label: "WI-FI" },
              { id: "cable", label: "CABLE" }
            ].map((f) => (
              <button
                key={f.id}
                onClick={() => {
                  setActiveFilter(f.id);
                  setCurrentPage(1);
                }}
                className={`px-3 py-1 text-xs border-2 font-bold cursor-pointer transition-colors ${
                  activeFilter === f.id
                    ? "bg-primary text-white border-primary"
                    : "bg-neutral-900 text-neutral-400 border-neutral-800 hover:border-neutral-600"
                }`}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>

        <p className="flex items-center gap-1.5 text-xs text-neutral-400 font-sans">
          <HelpCircle className="h-3.5 w-3.5 text-primary shrink-0" />
          Wi-Fi (95%) indica la calidad de la señal entre el aparato y el router; 95% es una señal excelente.
        </p>
      </div>

      <div className="flex items-center justify-between gap-3 text-xs font-mono text-neutral-500">
        <span>
          {filteredClients.length === 0
            ? "No hay aparatos que coincidan"
            : `Mostrando ${firstVisibleClient + 1}-${Math.min(firstVisibleClient + DEVICES_PER_PAGE, filteredClients.length)} de ${filteredClients.length}`}
        </span>
        {totalPages > 1 && <span>{totalPages} páginas</span>}
      </div>

      {/* Minimal Cards Grid (Clean layout, straight lines) */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {visibleClients.map((client) => {
          const alias = aliases[client.mac] || client.name || "Aparato";
          const isEditing = editingMac === client.mac;
          const dev = getDeviceDetails(client.name);
          const DeviceIcon = dev.icon;

          return (
            <div
              key={client.mac}
              className="border-2 border-neutral-800 hover:border-neutral-600 p-4 transition-colors space-y-3 relative group bg-black/50"
            >
              {/* Device Icon and State */}
              <div className="flex items-center justify-between border-b border-neutral-850 pb-2">
                <div className="flex items-center gap-2.5">
                  <div className="p-2 border border-neutral-800 bg-neutral-900 text-white">
                    <DeviceIcon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0">
                    {isEditing ? (
                      <div className="flex items-center gap-1.5">
                        <input
                          type="text"
                          value={tempAlias}
                          onChange={(e) => setTempAlias(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") handleSaveAlias(client.mac);
                            if (e.key === "Escape") setEditingMac(null);
                          }}
                          autoFocus
                          className="w-full min-w-0 bg-black border-2 border-primary px-2 py-1 text-sm font-bold text-white font-sans focus:outline-none"
                        />
                        <button
                          onClick={() => handleSaveAlias(client.mac)}
                          title="Guardar"
                          className="p-1.5 bg-primary hover:bg-primary-hover text-white border-2 border-primary cursor-pointer"
                        >
                          <Check className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => setEditingMac(null)}
                          title="Cancelar"
                          className="p-1.5 bg-neutral-800 hover:bg-neutral-700 text-neutral-300 border-2 border-neutral-700 cursor-pointer"
                        >
                          <X className="h-4 w-4" />
                        </button>
                      </div>
                    ) : (
                      <span
                        onClick={() => handleStartEdit(client.mac, alias)}
                        className="text-base font-bold text-white hover:text-primary transition-colors cursor-pointer truncate block"
                        title="Clic para editar"
                      >
                        {alias}
                      </span>
                    )}
                    <span className="text-xs text-neutral-500 font-sans">
                      {dev.connType}
                    </span>
                  </div>
                </div>

                <div className="flex items-center gap-1">
                  <span
                    className={`h-2.5 w-2.5 rounded-full ${client.active === false ? "bg-rose-500" : "bg-emerald-400 animate-pulse"}`}
                    title={client.active === false ? "Desconectado" : "Conectado"}
                    aria-label={client.active === false ? "Desconectado" : "Conectado"}
                  />
                  {!isEditing && (
                    <button
                      onClick={() => handleStartEdit(client.mac, alias)}
                      className="p-1 text-neutral-500 hover:text-white transition-colors cursor-pointer"
                      title="Cambiar alias"
                      aria-label="Cambiar alias"
                    >
                      <Edit3 className="h-3.5 w-3.5" />
                    </button>
                  )}
                  {aliases[client.mac] && (
                    <button
                      onClick={() => handleResetAlias(client.mac)}
                      className="p-1 text-neutral-500 hover:text-rose-400 transition-colors cursor-pointer"
                      title="Restaurar alias"
                      aria-label="Restaurar alias"
                    >
                      <RotateCcw className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              </div>

              {/* Minimal Telemetry Values (Border Left dividers) */}
              <div className="border-l-2 border-neutral-800 pl-4 pt-2 space-y-1.5 text-sm font-mono">
                <div className="flex justify-between gap-3">
                  <span className="text-neutral-500 text-sm">IP:</span>
                  <span className="text-white text-base font-bold">{client.ip}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-neutral-500 text-sm">HOST:</span>
                  <span className="text-neutral-300 text-base truncate max-w-xs">{client.name}</span>
                </div>
                <div className="flex justify-between gap-3">
                  <span className="text-neutral-500 text-sm">MAC:</span>
                  <span className="text-neutral-500 text-base">{client.mac}</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {totalPages > 1 && (
        <div className="flex flex-wrap items-center justify-center gap-2 border-t border-neutral-800 pt-4">
          <button
            onClick={() => setCurrentPage((page) => Math.max(1, page - 1))}
            disabled={visiblePage === 1}
            className="inline-flex items-center gap-1 border-2 border-neutral-800 bg-neutral-900 px-3 py-1.5 text-xs font-mono font-bold text-neutral-300 hover:border-neutral-600 disabled:cursor-not-allowed disabled:opacity-40"
            aria-label="Página anterior"
          >
            <ChevronLeft className="h-3.5 w-3.5" />
            Anterior
          </button>
          <span className="px-2 text-xs font-mono text-neutral-400">
            Página {visiblePage} de {totalPages}
          </span>
          <button
            onClick={() => setCurrentPage((page) => Math.min(totalPages, page + 1))}
            disabled={visiblePage === totalPages}
            className="inline-flex items-center gap-1 border-2 border-neutral-800 bg-neutral-900 px-3 py-1.5 text-xs font-mono font-bold text-neutral-300 hover:border-neutral-600 disabled:cursor-not-allowed disabled:opacity-40"
            aria-label="Página siguiente"
          >
            Siguiente
            <ChevronRight className="h-3.5 w-3.5" />
          </button>
        </div>
      )}
    </div>
  );
}
