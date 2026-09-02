import React, { useState } from "react";
import { Shield, Wifi, Globe, Share2, Server, HelpCircle, Lock, ExternalLink, X } from "lucide-react";
import { mockSecurity } from "@/data/mockData";

export function CapabilitiesGrid({ capabilities = {} }) {
  const [selectedCap, setSelectedCap] = useState(null);

  const getCapabilityMeta = (key) => {
    switch (key) {
      case "device":
        return {
          label: "Device Identity",
          path: "/v0/device",
          icon: Server,
          desc: "Vendor, model, hardware & firmware provenance verification."
        };
      case "status":
        return {
          label: "System Status & WAN",
          path: "/v0/status",
          icon: Globe,
          desc: "WAN link state, reachability check, and uptime telemetry."
        };
      case "clients":
        return {
          label: "DHCP Clients",
          path: "/v0/clients",
          icon: Share2,
          desc: "Active local IP/MAC leases assigned to network clients."
        };
      case "wireless_security":
        return {
          label: "Wireless Security",
          path: "/v0/security/wireless",
          icon: Wifi,
          desc: mockSecurity.wireless.description
        };
      case "dmz":
        return {
          label: "DMZ (Demilitarized Zone)",
          path: "/v0/security/dmz",
          icon: Shield,
          desc: mockSecurity.dmz.description
        };
      case "forwarding":
        return {
          label: "Port Forwarding Rules",
          path: "/v0/security/forwarding",
          icon: ExternalLink,
          desc: mockSecurity.forwarding.description
        };
      case "wps":
        return {
          label: "WPS (Wi-Fi Protected)",
          path: "/v0/security/wps",
          icon: Lock,
          desc: mockSecurity.wps.description
        };
      case "upnp":
        return {
          label: "UPnP (Plug & Play)",
          path: "/v0/security/upnp",
          icon: HelpCircle,
          desc: mockSecurity.upnp.description
        };
      case "remote_management":
        return {
          label: "Remote Management",
          path: "/v0/security/remote-management",
          icon: Server,
          desc: mockSecurity.remote_management.description
        };
      default:
        return {
          label: key,
          path: `/v0/${key}`,
          icon: Shield,
          desc: "Standard observation capability."
        };
    }
  };

  return (
    <div className="space-y-4 font-sans text-white">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {Object.entries(capabilities).map(([key, state]) => {
          const meta = getCapabilityMeta(key);
          const Icon = meta.icon;

          return (
            <div
              key={key}
              onClick={() => setSelectedCap(key)}
              className="cursor-pointer border-2 border-neutral-800 bg-neutral-900 hover:border-primary p-4 transition-colors space-y-2"
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="p-1.5 bg-black border border-neutral-800">
                    <Icon className="h-4 w-4 text-white" />
                  </div>
                  <h5 className="font-bold text-xs uppercase text-white">{meta.label}</h5>
                </div>
                <span className="text-xs font-mono px-2 py-0.5 border border-neutral-800 bg-black text-neutral-300 uppercase font-semibold">
                  {state.replace(/_/g, " ")}
                </span>
              </div>

              <p className="text-xs text-neutral-400 line-clamp-2">
                {meta.desc}
              </p>

              <div className="flex items-center justify-between text-xs font-mono text-neutral-400 border-t border-neutral-800 pt-2">
                <span>{meta.path}</span>
                <span className="text-primary font-bold">Ver &rarr;</span>
              </div>
            </div>
          );
        })}
      </div>

      {/* Modal detail */}
      {selectedCap && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-xs">
          <div className="w-full max-w-lg border-2 border-neutral-700 bg-neutral-950 p-6 shadow-2xl space-y-4 text-white">
            <div className="flex items-center justify-between pb-3 border-b-2 border-neutral-800">
              <h3 className="font-black text-sm uppercase text-white">
                {getCapabilityMeta(selectedCap).label}
              </h3>
              <button
                onClick={() => setSelectedCap(null)}
                className="text-neutral-400 hover:text-white p-1 cursor-pointer"
              >
                <X className="h-4 w-4" />
              </button>
            </div>

            <div className="space-y-3 text-xs">
              <div>
                <span className="text-neutral-400 block mb-1 font-mono uppercase">Endpoint:</span>
                <code className="bg-black border border-neutral-800 px-2 py-1 font-mono text-primary font-bold block">
                  GET {getCapabilityMeta(selectedCap).path}
                </code>
              </div>

              <div>
                <span className="text-neutral-400 block mb-1 font-mono uppercase">Descripción:</span>
                <p className="text-neutral-300 leading-relaxed">{getCapabilityMeta(selectedCap).desc}</p>
              </div>

              <div>
                <span className="text-neutral-400 block mb-1 font-mono uppercase">Estado observado:</span>
                <div className="bg-black border border-neutral-800 p-2.5 font-mono text-xs text-neutral-300">
                  {capabilities[selectedCap] === "verified" && "✓ Verificado contra hardware real TL-WR841N."}
                  {capabilities[selectedCap] === "absent" && "ℹ Característica ausente en este firmware (HTTP 404)."}
                  {capabilities[selectedCap] === "unsupported_or_unverified" && "⏳ Capacidad reservada aún no observada en capturas."}
                  {capabilities[selectedCap] === "unavailable" && "⚠ Servicio no disponible temporalmente (HTTP 503)."}
                </div>
              </div>
            </div>

            <div className="pt-2 flex justify-end">
              <button
                onClick={() => setSelectedCap(null)}
                className="bg-neutral-900 border-2 border-neutral-700 text-white font-mono text-xs px-4 py-2 uppercase font-bold hover:bg-neutral-800 cursor-pointer"
              >
                Cerrar
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
