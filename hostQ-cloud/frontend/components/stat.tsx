import { cn } from "@/lib/utils";
import type { LucideIcon } from "lucide-react";

// Compact stat card. Used on dashboards. Premium feel = small label, big
// number, optional trend underneath. No gradient backgrounds — just clean
// borders and tight typography.
export function Stat({
  label,
  value,
  sub,
  icon: Icon,
  className,
}: {
  label: string;
  value: React.ReactNode;
  sub?: React.ReactNode;
  icon?: LucideIcon;
  className?: string;
}) {
  return (
    <div className={cn("rounded-lg border border-border bg-surface px-4 py-3 shadow-sm", className)}>
      <div className="flex items-center gap-1.5 text-[10.5px] font-semibold uppercase tracking-wider text-muted">
        {Icon && <Icon className="h-3 w-3 opacity-60" />}
        <span>{label}</span>
      </div>
      <div className="mt-1 text-xl font-semibold tracking-tight leading-tight">{value}</div>
      {sub && <div className="mt-0.5 text-xs text-muted">{sub}</div>}
    </div>
  );
}
