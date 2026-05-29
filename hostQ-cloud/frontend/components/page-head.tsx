import { cn } from "@/lib/utils";

// Standard page header. Title left, actions right. Used on every list page
// for a consistent rhythm.
export function PageHead({
  title,
  description,
  actions,
  className,
}: {
  title: string;
  description?: string;
  actions?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex flex-wrap items-end justify-between gap-3 mb-4", className)}>
      <div className="min-w-0">
        <h1 className="text-[17px] font-semibold tracking-tight text-balance">{title}</h1>
        {description && <p className="text-xs text-muted mt-0.5 max-w-prose">{description}</p>}
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </div>
  );
}
