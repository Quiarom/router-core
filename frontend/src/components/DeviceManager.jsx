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
  ChevronRight
} from "lucide-react";

const DEVICES_PER_PAGE = 6;

export function DeviceManager({ clients = [], state: _state = "verified" }) {
  const [aliases, setAliases] = useState(() => {
    try {
      const saved = localStorage.getItem("router_core_device_aliases");
      if (saved) return JSON.parse(saved);
    } catch (e) {
      console.error(e);
    }
    return {
      "00:11:22:33:44:55": "Computadora Portátil",
      "AA:BB:CC:11:22:33": "Teléfono Personal",
      "10:20:30:40:50:60": "Smart TV Sala"
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
        type: "Teléfono",
        icon: Smartphone,
        connType: "Wi-Fi (Señal 95%)",
        isWifi: true
      };
    }
    if (lower.includes("tv") || lower.includes("smart") || lower.includes("tele") || lower.includes("roku")) {
      return {
        type: "Smart TV",
        icon: Tv,
        connType: "Cable de red o Wi-Fi",
        isWifi: false
      };
    }
    if (lower.includes("laptop") || lower.includes("macbook") || lower.includes("notebook") || lower.includes("portatil")) {
      return {
        type: "Portátil",
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
          <div className="flex items-center gap-2.5">
            <h2 className="text-xl sm:text-2xl font-black uppercase tracking-tight text-white font-sans">
              Aparatos en tu Casa
            </h2>
            <span className="font-mono text-xs font-bold text-primary bg-primary/10 border border-primary/30 px-2 py-0.5">
              {clients.length}
            </span>
          </div>
        </div>

        {/* Filter segment */}
        <div className="flex items-center gap-1.5 self-start sm:self-auto font-mono text-xs">
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
              className={`px-3 py-1.5 border-2 font-bold uppercase transition-colors cursor-pointer ${
                activeFilter === f.id
                  ? "bg-primary text-white border-primary"
                  : "bg-neutral-900 text-neutral-400 border-neutral-800 hover:border-neutral-700"
              }`}
            >
              {f.label}
            </button>
          ))}
        </div>
      </div>

      {/* Search Bar */}
      <div className="relative">
        <Search className="h-4 w-4 text-neutral-500 absolute left-3.5 top-1/2 -translate-y-1/2" />
        <input
          type="text"
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            setCurrentPage(1);
          }}
          placeholder="BUSCAR POR ALIAS, IP O MAC..."
          className="w-full bg-black border-2 border-neutral-800 pl-10 pr-4 py-2.5 text-xs font-mono text-white placeholder-neutral-500 focus:border-primary focus:outline-none transition-colors"
        />
      </div>

      {/* Devices Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {visibleClients.map((client) => {
          const alias = aliases[client.mac] || client.name || "Dispositivo";
          const isEditing = editingMac === client.mac;
          const dev = getDeviceDetails(client.name);
          const DeviceIcon = dev.icon;

          return (
            <div
              key={client.mac}
              className="border-2 border-neutral-800 hover:border-neutral-600 p-4 transition-colors space-y-3 relative group bg-black/40"
            >
              {/* Device Icon and State */}
              <div className="flex items-center justify-between pb-2 border-b border-neutral-800/80">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="h-9 w-9 bg-neutral-900 border border-neutral-800 flex items-center justify-center text-neutral-300 shrink-0">
                    <DeviceIcon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0 flex-1">
                    {isEditing ? (
                      <div className="flex items-center gap-1 font-mono">
                        <input
                          type="text"
                          value={tempAlias}
                          onChange={(e) => setTempAlias(e.target.value)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") handleSaveAlias(client.mac);
                            if (e.key === "Escape") setEditingMac(null);
                          }}
                          autoFocus
                          className="w-full bg-black border border-primary px-2 py-0.5 text-xs text-white focus:outline-none"
                        />
                        <button
                          onClick={() => handleSaveAlias(client.mac)}
                          title="Guardar"
                          className="p-1 bg-primary text-white hover:bg-primary-hover cursor-pointer"
                        >
                          <Check className="h-3 w-3" />
                        </button>
                        <button
                          onClick={() => setEditingMac(null)}
                          title="Cancelar"
                          className="p-1 bg-neutral-800 text-neutral-400 hover:text-white cursor-pointer"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <span
                          onClick={() => handleStartEdit(client.mac, alias)}
                          className="text-sm font-bold text-white hover:text-primary transition-colors cursor-pointer truncate font-mono"
                          title="Clic para editar nombre"
                        >
                          {alias}
                        </span>
                        <button
                          onClick={() => handleStartEdit(client.mac, alias)}
                          className="opacity-0 group-hover:opacity-100 text-neutral-500 hover:text-primary transition-opacity p-0.5 cursor-pointer"
                          title="Cambiar nombre"
                        >
                          <Edit3 className="h-3 w-3" />
                        </button>
                      </div>
                    )}
                    <span className="text-[11px] text-neutral-500 block mt-0.5 font-mono">
                      {dev.connType}
                    </span>
                  </div>
                </div>

                {/* Active indicator & reset alias */}
                <div className="flex items-center gap-1.5 shrink-0">
                  <span
                    className={`h-2 w-2 rounded-full ${client.active === false ? "bg-neutral-600" : "bg-emerald-500 animate-pulse"}`}
                    title={client.active === false ? "Desconectado" : "Conectado"}
                  />
                  {aliases[client.mac] && (
                    <button
                      onClick={() => handleResetAlias(client.mac)}
                      className="text-neutral-500 hover:text-neutral-300 transition-colors p-1 cursor-pointer"
                      title="Restablecer nombre original"
                    >
                      <RotateCcw className="h-3 w-3" />
                    </button>
                  )}
                </div>
              </div>

              {/* Technical details: Brutalist border-l-2 */}
              <div className="grid grid-cols-2 gap-2 text-xs font-mono text-neutral-400 pt-1 border-t border-neutral-800/60">
                <div>
                  <span className="text-neutral-500 text-[10px] block">IP LOCAL</span>
                  <span className="text-white text-xs font-bold">{client.ip}</span>
                </div>
                <div>
                  <span className="text-neutral-500 text-[10px] block">MAC</span>
                  <span className="text-neutral-300 text-xs truncate block" title={client.mac}>
                    {client.mac}
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between border-t-2 border-neutral-800 pt-4 text-xs font-mono text-neutral-400">
          <span>
            MOSTRANDO {firstVisibleClient + 1}-{Math.min(firstVisibleClient + DEVICES_PER_PAGE, filteredClients.length)} DE {filteredClients.length}
          </span>
          <div className="flex items-center gap-1.5">
            <button
              onClick={() => setCurrentPage((p) => Math.max(1, p - 1))}
              disabled={visiblePage === 1}
              className="p-1.5 border-2 border-neutral-800 bg-neutral-900 hover:bg-neutral-800 text-white disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </button>
            <span className="px-2 font-bold text-white">
              {visiblePage} / {totalPages}
            </span>
            <button
              onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))}
              disabled={visiblePage === totalPages}
              className="p-1.5 border-2 border-neutral-800 bg-neutral-900 hover:bg-neutral-800 text-white disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed"
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

export default DeviceManager;
