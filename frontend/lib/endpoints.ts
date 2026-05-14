export const backendURL =
  process.env.NEXT_PUBLIC_BACKEND_URL ?? "http://localhost:8080";
export const timelineWSURL =
  process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080/ws/timeline";
export const explorerURL = process.env.NEXT_PUBLIC_OG_EXPLORER_URL ?? "";

export function resolveImageSrc(imageRef: string): string {
  if (!imageRef) return "";
  if (imageRef.startsWith("data:") || imageRef.startsWith("http")) {
    return imageRef;
  }
  return `${backendURL}/storage?pointer=${encodeURIComponent(imageRef)}`;
}

export function resolveStoragePointerSrc(pointer: string): string {
  if (!pointer) return "";
  if (pointer.startsWith("http")) {
    return pointer;
  }
  return `${backendURL}/storage?pointer=${encodeURIComponent(pointer)}`;
}
