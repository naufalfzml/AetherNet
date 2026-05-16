import { existsSync, readFileSync } from "node:fs";
import path from "node:path";

function applyEnvFile(filePath: string) {
  if (!existsSync(filePath)) {
    return false;
  }

  const raw = readFileSync(filePath, "utf8");
  for (const line of raw.split(/\r?\n/)) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const separator = trimmed.indexOf("=");
    if (separator <= 0) {
      continue;
    }

    const key = trimmed.slice(0, separator).trim();
    if (!key || process.env[key] !== undefined) {
      continue;
    }

    const value = trimmed.slice(separator + 1).trim().replace(/^['"]|['"]$/g, "");
    process.env[key] = value;
  }
  return true;
}

export function loadEnvFile(startDir = process.cwd()) {
  let currentDir = path.resolve(startDir);

  while (true) {
    const filePath = path.join(currentDir, ".env");
    if (applyEnvFile(filePath)) {
      return filePath;
    }

    const parentDir = path.dirname(currentDir);
    if (parentDir === currentDir) {
      return;
    }
    currentDir = parentDir;
  }
}
