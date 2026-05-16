function normalizeURL(value: string | undefined): string {
  return (value ?? "").trim().replace(/\/+$/, "");
}

export const backendURL = normalizeURL(process.env.NEXT_PUBLIC_BACKEND_URL);
export const timelineWSURL = normalizeURL(process.env.NEXT_PUBLIC_WS_URL);
export const explorerURL = process.env.NEXT_PUBLIC_OG_EXPLORER_URL ?? "";

export const backendURLConfigured = backendURL !== "";
export const timelineWSURLConfigured = timelineWSURL !== "";

export function resolveBackendPath(path: string): string {
  if (!backendURLConfigured) {
    throw new Error(
      "NEXT_PUBLIC_BACKEND_URL is not configured. Set the deployed backend origin in frontend environment variables.",
    );
  }
  return `${backendURL}${path}`;
}

export function resolveImageSrc(imageRef: string): string {
  if (!imageRef) return "";
  if (imageRef.startsWith("data:") || imageRef.startsWith("http")) {
    return imageRef;
  }
  return resolveBackendPath(
    `/storage?pointer=${encodeURIComponent(imageRef)}`,
  );
}

export function resolveStoragePointerSrc(pointer: string): string {
  if (!pointer) return "";
  if (pointer.startsWith("http")) {
    return pointer;
  }
  return resolveBackendPath(`/storage?pointer=${encodeURIComponent(pointer)}`);
}
