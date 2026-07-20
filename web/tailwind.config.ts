import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{js,ts,jsx,tsx,mdx}",
    "./components/**/*.{js,ts,jsx,tsx,mdx}",
    "./lib/**/*.{js,ts,jsx,tsx,mdx}",
  ],
  theme: {
    extend: {
      colors: {
        bg: "#0a0a0b",
        surface: "#131316",
        border: "#232327",
        text: {
          primary: "#e4e4e7",
          secondary: "#9a9aa2",
          muted: "#5f5f68",
        },
        accent: {
          DEFAULT: "#3b82f6",
          hover: "#60a5fa",
        },
        danger: "#ef4444",
        warn: "#f59e0b",
        ok: "#22c55e",
      },
      fontFamily: {
        mono: ["JetBrains Mono", "monospace"],
        sans: ["Inter", "sans-serif"],
      },
    },
  },
  plugins: [],
};

export default config;