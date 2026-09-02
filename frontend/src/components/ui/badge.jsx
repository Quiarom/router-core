import React from "react";
import { cva } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center rounded-md border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        default: "border-transparent bg-slate-800 text-slate-100",
        secondary: "border-transparent bg-slate-700 text-slate-200",
        outline: "text-slate-300 border-slate-700",
        verified: "border-emerald-500/30 bg-emerald-500/10 text-emerald-400",
        absent: "border-slate-600/40 bg-slate-800/40 text-slate-400",
        unverified: "border-amber-500/30 bg-amber-500/10 text-amber-400",
        unavailable: "border-rose-500/30 bg-rose-500/10 text-rose-400",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

export function Badge({ className, variant, status, ...props }) {
  // Map contract capability states directly to variant if provided
  let activeVariant = variant;
  if (status) {
    if (status === "verified") activeVariant = "verified";
    else if (status === "absent") activeVariant = "absent";
    else if (status === "unsupported_or_unverified" || status === "unverified") activeVariant = "unverified";
    else if (status === "unavailable") activeVariant = "unavailable";
  }

  return (
    <div className={cn(badgeVariants({ variant: activeVariant }), className)} {...props} />
  );
}
