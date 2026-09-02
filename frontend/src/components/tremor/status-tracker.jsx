import React from "react";
import { cn } from "@/lib/utils";

export function StatusTracker({ items = [], className }) {
  const getStatusColor = (status) => {
    switch (status) {
      case "verified":
        return "bg-emerald-500 hover:bg-emerald-400";
      case "absent":
        return "bg-slate-700 hover:bg-slate-600";
      case "unverified":
      case "unsupported_or_unverified":
        return "bg-amber-500 hover:bg-amber-400";
      case "unavailable":
        return "bg-rose-500 hover:bg-rose-400";
      default:
        return "bg-slate-600";
    }
  };

  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex h-3 w-full gap-1 overflow-hidden rounded-md bg-slate-800/60 p-0.5">
        {items.map((item, index) => (
          <div
            key={index}
            title={`${item.name}: ${item.status}`}
            className={cn(
              "h-full flex-1 rounded-sm transition-all cursor-pointer",
              getStatusColor(item.status)
            )}
          />
        ))}
      </div>
      <div className="flex flex-wrap items-center justify-between text-[11px] text-slate-400">
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-emerald-500" />
          Verified ({items.filter((i) => i.status === "verified").length})
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-slate-600" />
          Absent ({items.filter((i) => i.status === "absent").length})
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-amber-500" />
          Unverified ({items.filter((i) => i.status === "unverified" || i.status === "unsupported_or_unverified").length})
        </span>
        <span className="flex items-center gap-1.5">
          <span className="h-2 w-2 rounded-full bg-rose-500" />
          Unavailable ({items.filter((i) => i.status === "unavailable").length})
        </span>
      </div>
    </div>
  );
}
