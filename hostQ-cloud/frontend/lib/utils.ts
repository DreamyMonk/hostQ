import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

// cn is the shadcn-standard className combiner. Resolves conflicting Tailwind
// classes deterministically so children can override their parent.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
