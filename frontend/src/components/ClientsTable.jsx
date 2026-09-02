import React from "react";
import { Laptop, Smartphone, Tv, Monitor, Shield } from "lucide-react";
import { Badge } from "@/components/ui/badge";

export function ClientsTable({ clients = [], state = "verified" }) {
  const getDeviceIcon = (name = "") => {
    const lower = name.toLowerCase();
    if (lower.includes("phone") || lower.includes("pixel") || lower.includes("iphone")) {
      return <Smartphone className="h-4 w-4 text-cyan-400" />;
    }
    if (lower.includes("tv") || lower.includes("smart")) {
      return <Tv className="h-4 w-4 text-purple-400" />;
    }
    if (lower.includes("laptop") || lower.includes("macbook")) {
      return <Laptop className="h-4 w-4 text-emerald-400" />;
    }
    return <Monitor className="h-4 w-4 text-slate-400" />;
  };

  return (
    <div className="overflow-hidden rounded-xl border border-slate-800 bg-slate-900/40">
      <div className="flex items-center justify-between border-b border-slate-800/80 px-5 py-3.5 bg-slate-900/60">
        <div>
          <h4 className="text-sm font-semibold text-white">Active DHCP Leases</h4>
          <p className="text-xs text-slate-400">Devices currently recognized by the router's DHCP server</p>
        </div>
        <Badge status={state}>{state.toUpperCase()}</Badge>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs">
          <thead className="border-b border-slate-850 bg-slate-900/80 text-slate-400 uppercase tracking-wider font-mono text-[11px]">
            <tr>
              <th className="px-5 py-3 font-medium">Device Name</th>
              <th className="px-5 py-3 font-medium">IP Address</th>
              <th className="px-5 py-3 font-medium">MAC Address</th>
              <th className="px-5 py-3 font-medium">Lease Time</th>
              <th className="px-5 py-3 font-medium">Trust Provenance</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-850/60 text-slate-300">
            {clients.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-5 py-8 text-center text-slate-500">
                  No active client leases found.
                </td>
              </tr>
            ) : (
              clients.map((client, idx) => (
                <tr key={client.mac || idx} className="hover:bg-slate-800/30 transition-colors">
                  <td className="px-5 py-3 font-medium text-white flex items-center gap-2.5">
                    {getDeviceIcon(client.name)}
                    <span>{client.name || "Unknown Device"}</span>
                  </td>
                  <td className="px-5 py-3 font-mono text-emerald-400">{client.ip}</td>
                  <td className="px-5 py-3 font-mono text-slate-400">{client.mac}</td>
                  <td className="px-5 py-3 font-mono text-slate-300">
                    <span className="rounded bg-slate-800 px-2 py-0.5 text-[11px]">
                      {client.lease}
                    </span>
                  </td>
                  <td className="px-5 py-3">
                    <span className="inline-flex items-center gap-1 text-[11px] text-slate-400">
                      <Shield className="h-3 w-3 text-slate-500" />
                      router:dhcp
                    </span>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
