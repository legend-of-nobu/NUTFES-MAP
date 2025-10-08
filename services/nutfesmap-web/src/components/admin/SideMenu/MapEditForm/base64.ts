const DEFAULT_IMAGE_MIME = "image/png";

function guessImageMimeFromBase64(trimmedBase64: string): string {
  // 代表的なヘッダで簡易判定（先頭のみチェック）
  if (!trimmedBase64) return DEFAULT_IMAGE_MIME;
  if (trimmedBase64.startsWith("PHN2") || trimmedBase64.startsWith("PD94")) return "image/svg+xml"; // "<svg" or "<?xml"
  if (trimmedBase64.startsWith("iVBORw0KGgo")) return "image/png";
  if (trimmedBase64.startsWith("/9j/")) return "image/jpeg";
  if (trimmedBase64.startsWith("R0lGODdh") || trimmedBase64.startsWith("R0lGODlh")) return "image/gif";
  if (trimmedBase64.startsWith("UklGR")) return "image/webp";
  if (trimmedBase64.startsWith("Qk")) return "image/bmp";
  if (trimmedBase64.startsWith("AAABAAEAEBAQ")) return "image/x-icon";
  return DEFAULT_IMAGE_MIME;
}

// Base64（ヘッダ無しのプレーン）→ dataURL（<img src=...>で使える形）に包む
export function base64PlainToDataUrl(b64: string, mime?: string): string {
  if (!b64 || typeof b64 !== "string") return "";
  const trimmed = b64.replace(/\s+/g, "");
  // ゆるめのBase64判定（必要十分）
  const maybeBase64 = /^[A-Za-z0-9+/]+={0,2}$/.test(trimmed);
  if (!maybeBase64) return b64;

  const resolvedMime = mime ?? guessImageMimeFromBase64(trimmed);
  return `data:${resolvedMime};base64,${trimmed}`;
}

// プレビュー用URLを安全に作る: nullはnullのまま、http(s)/dataはそのまま、その他はBase64とみなす
export function toPreviewUrl(src: string | null | undefined, mime?: string) {
  if (!src) return null;
  if (
    src.startsWith("data:") ||
    src.startsWith("http://") ||
    src.startsWith("https://") ||
    src.startsWith("blob:")
  ) {
    return src;
  }
  return base64PlainToDataUrl(src, mime);
}
