// Base64（ヘッダ無しのプレーン）→ dataURL（<img src=...>で使える形）に包む
export function base64PlainToDataUrl(b64: string, mime = "image/png"): string {
  if (!b64 || typeof b64 !== "string") return "";
  const trimmed = b64.replace(/\s+/g, "");
  // ゆるめのBase64判定（必要十分）
  const maybeBase64 = /^[A-Za-z0-9+/]+={0,2}$/.test(trimmed);
  return maybeBase64 ? `data:${mime};base64,${trimmed}` : b64;
}

// プレビュー用URLを安全に作る: nullはnullのまま、http(s)/dataはそのまま、その他はBase64とみなす
export function toPreviewUrl(src: string | null | undefined, mime = "image/png") {
  if (!src) return null;
  if (src.startsWith("data:") || src.startsWith("http://") || src.startsWith("https://")) {
    return src;
  }
  return base64PlainToDataUrl(src, mime);
}
