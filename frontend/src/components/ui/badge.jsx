import React from "react";
import { cva } from "class-variance-authority";
import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center border px-2 py-0.5 text-xs font-mono font-bold uppercase transition-colors focus:outline-none",
  {
    variants: {
      variant: {
        default: "border-neutral-800 bg-neutral-900 text-neutral-300",
        secondary: "border-neutral-800 bg-neutral-800 text-neutral-400",
        outline: "text-neutral-300 border-neutral-700",
        verified: "border-emerald-500/40 bg-emerald-500/10 text-emerald-400",
        absent: "border-neutral-800 bg-neutral-900 text-neutral-500",
        unverified: "border-amber-500/40 bg-amber-500/10 text-amber-400",
        unavailable: "border-rose-500/40 bg-rose-500/10 text-rose-400",
        primary: "border-primary bg-primary/10 text-primary",
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
