import type { Config } from "tailwindcss";

// Token-driven Tailwind config. All colors reference CSS variables defined
// in app/globals.css so light/dark themes share the same component code.
export default {
  darkMode: "class",
  content: ["./app/**/*.{ts,tsx}", "./components/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        canvas: "hsl(var(--canvas))",
        surface: "hsl(var(--surface))",
        elevated: "hsl(var(--elevated))",
        border: "hsl(var(--border))",
        "border-strong": "hsl(var(--border-strong))",
        ink: "hsl(var(--ink))",
        muted: "hsl(var(--muted))",
        faint: "hsl(var(--faint))",
        accent: {
          DEFAULT: "hsl(var(--accent))",
          fg: "hsl(var(--accent-fg))",
        },
        success: "hsl(var(--success))",
        warning: "hsl(var(--warning))",
        danger: "hsl(var(--danger))",
      },
      fontFamily: {
        sans: ["Inter", "ui-sans-serif", "system-ui", "sans-serif"],
        mono: ["ui-monospace", "SFMono-Regular", "Consolas", "monospace"],
      },
      borderRadius: {
        DEFAULT: "0.5rem",
        lg: "0.625rem",
        xl: "0.75rem",
      },
      boxShadow: {
        sm: "0 1px 2px hsl(0 0% 0% / 0.04)",
        DEFAULT: "0 1px 3px hsl(0 0% 0% / 0.06), 0 1px 2px hsl(0 0% 0% / 0.04)",
        lg: "0 10px 30px hsl(0 0% 0% / 0.10), 0 4px 8px hsl(0 0% 0% / 0.04)",
      },
    },
  },
  plugins: [],
} satisfies Config;
