import React from "react";
import { cva } from "class-variance-authority";
import { cn } from "@/lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center whitespace-nowrap text-xs font-mono font-bold uppercase transition-colors focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 cursor-pointer",
  {
    variants: {
      variant: {
        default: "bg-primary text-white border-2 border-primary hover:bg-primary-hover",
        destructive: "bg-rose-600 text-white border-2 border-rose-600 hover:bg-rose-700",
        outline: "border-2 border-neutral-700 bg-neutral-900 text-neutral-300 hover:border-neutral-500 hover:text-white",
        secondary: "bg-neutral-800 text-white border-2 border-neutral-700 hover:bg-neutral-700",
        ghost: "hover:bg-neutral-850 hover:text-white text-neutral-400",
        link: "text-primary underline-offset-4 hover:underline",
      },
      size: {
        default: "h-9 px-4 py-2",
        sm: "h-8 px-3 text-xs",
        lg: "h-11 px-6 text-sm",
        icon: "h-9 w-9",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
);

export function Button({ className, variant, size, ...props }) {
  return (
    <button
      className={cn(buttonVariants({ variant, size, className }))}
      {...props}
    />
  );
}
