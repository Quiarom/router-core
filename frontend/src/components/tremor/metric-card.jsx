import React from "react";
import { cn } from "@/lib/utils";

export function MetricCard({
  title,
  metric,
  subtext,
  badgeText,
  badgeVariant = "default",
  icon: Icon,
  className,
}) {
  return (
    <div
      className={cn(
        "relative overflow-hidden rounded-xl border border-slate-800 bg-slate-900/60 p-5 shadow-sm backdrop-blur-sm transition-all hover:border-slate-700/80",
        className
      )}
    >
      <div className="flex items-center justify-between">
        <span className="text-xs font-medium uppercase tracking-wider text-slate-400">
          {title}
        </span>
        {Icon && (
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-slate-800/80 text-slate-300">
            <Icon className="h-4 w-4" />
          </div>
        )}
      </div>

      <div className="mt-3 flex items-baseline gap-2">
        <span className="text-2xl font-bold tracking-tight text-white font-mono">
          {metric}
        </span>
        {badgeText && (
          <span
            className={cn(
              "inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium",
              badgeVariant === "success" && "bg-emerald-500/15 text-emerald-400 border border-emerald-500/20",
              badgeVariant === "warning" && "bg-amber-500/15 text-amber-400 border border-amber-500/20",
              badgeVariant === "danger" && "bg-rose-500/15 text-rose-400 border border-rose-500/20",
              badgeVariant === "default" && "bg-slate-800 text-slate-300 border border-slate-700"
            )}
          >
            {badgeText}
          </span>
        )}
      </div>

      {subtext && (
        <p className="mt-1 text-xs text-slate-400 flex items-center gap-1">
          {subtext}
        </p>
      )}
    </div>
  );
}
