import type { Config } from "tailwindcss";

const config: Config = {
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./lib/**/*.{ts,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        ink: "#14110f",
        paper: "#f4f0e8",
        copper: "#b75f3a",
        moss: "#516b4f",
        signal: "#2f7de1",
      },
      boxShadow: {
        line: "0 1px 0 rgba(20,17,15,0.12)",
      },
    },
  },
  plugins: [],
};

export default config;
