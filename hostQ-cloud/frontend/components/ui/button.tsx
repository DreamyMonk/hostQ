"use client";

import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";
import { cn } from "@/lib/utils";

// shadcn-style button. Variants tuned for the premium SaaS aesthetic:
// neutral default, ink primary, subtle ghost, danger reserved for destructive.
const buttonVariants = cva(
  "inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium ring-offset-canvas transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 focus-visible:ring-offset-1 disabled:pointer-events-none disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0",
  {
    variants: {
      variant: {
        default:
          "bg-surface border border-border text-ink hover:bg-elevated hover:border-border-strong",
        primary:
          "bg-ink text-surface hover:bg-ink/90",
        brand:
          "bg-accent text-accent-fg hover:bg-accent/90",
        ghost:
          "text-muted hover:bg-elevated hover:text-ink",
        danger:
          "border border-danger/30 text-danger hover:bg-danger/10",
        link:
          "text-accent underline-offset-4 hover:underline",
      },
      size: {
        default: "h-8 px-3 text-[13px]",
        sm: "h-7 px-2.5 text-xs",
        lg: "h-9 px-4",
        icon: "h-8 w-8",
      },
    },
    defaultVariants: { variant: "default", size: "default" },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : "button";
    return <Comp className={cn(buttonVariants({ variant, size, className }))} ref={ref} {...props} />;
  }
);
Button.displayName = "Button";
