import React, { useEffect, useMemo, useState } from "react";
import { MapNameForm } from "./MapNameForm";
import { MapImageUploadForm } from "./MapImageUploadForm";
import PreviewMapImage from "./PreviewMapImage";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import "../Style.css";
import "../Image.css";
import { toPreviewUrl } from "./base64";

export type MapEditFormProps = {
  onClose: () => void;
  mapId: string;                         // 編集対象のMap ID（必須）
  initialName?: string;                  // 初期表示用マップ名
  initialImageUrl?: string | null;       // 初期プレビュー用（dataURL or 署名URL or plain Base64）
  onSaved?: (updated: {
    id: string;
    name: string;
    imageData?: string | null;           // ★API仕様に合わせて“ヘッダ無しBase64”で返す
  }) => void;
};

// 画像ファイル -> Base64（dataURLヘッダを除いたプレーン）
async function fileToBase64Plain(file: File): Promise<string> {
  const dataUrl = await new Promise<string>((resolve, reject) => {
    const r = new FileReader();
    r.onload = () => resolve(String(r.result));
    r.onerror = reject;
    r.readAsDataURL(file);
  });
  const comma = dataUrl.indexOf(",");
  return comma >= 0 ? dataUrl.slice(comma + 1) : dataUrl;
}

// dataURL → plain Base64（他はそのまま返す）
function dataUrlToPlainBase64(src: string | null | undefined): string | null {
  if (!src) return null;
  if (!src.startsWith("data:")) return src; // 既にplainの可能性 or URL
  return src.replace(/^data:.*;base64,/, "");
}

export const MapEditForm: React.FC<MapEditFormProps> = ({
  onClose,
  mapId,
  initialName = "",
  initialImageUrl = null,
  onSaved,
}) => {
  const [mapName, setMapName] = useState(initialName);
  const [mapImage, setMapImage] = useState<File | null>(null);
  const [saving, setSaving] = useState(false);

  // プレビューURL: File優先、なければ initialImageUrl を安全化
  const previewUrl = useMemo(() => {
    if (mapImage) return URL.createObjectURL(mapImage);
    return toPreviewUrl(initialImageUrl);
  }, [mapImage, initialImageUrl]);

  // ObjectURLの掃除
  useEffect(() => {
    return () => {
      if (previewUrl && previewUrl.startsWith("blob:")) {
        URL.revokeObjectURL(previewUrl);
      }
    };
  }, [previewUrl]);

  const handleSave = async () => {
    // 🧱 バリデーション
    if (!mapId) {
      alert("編集対象のマップが選択されていません。");
      return;
    }
    if (!mapName && !mapImage && !initialImageUrl) {
      alert("マップ名か画像のどちらかは入力してください。");
      return;
    }

    try {
      setSaving(true);

      // OAS: PATCH /maps/{mapId} の部分更新（必要な項目のみ送る）
      const payload: Record<string, any> = {};
      if (mapName && mapName !== initialName) payload.name = mapName;
      if (mapImage) payload.imageData = await fileToBase64Plain(mapImage); // format: byte（Base64, ヘッダ無し）

      // 変更がなければ即時成功扱い（onSavedへは“plain Base64”で返す）
      if (Object.keys(payload).length === 0) {
        onSaved?.({
          id: mapId,
          name: mapName,
          imageData: dataUrlToPlainBase64(initialImageUrl), // dataURL→plain
        });
        return;
      }

      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/maps/${encodeURIComponent(mapId)}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          credentials: "include", // Cookie 認証想定
          body: JSON.stringify(payload),
        }
      );

      if (!res.ok) {
        const text = await res.text();
        throw new Error(`PATCH /maps/${mapId} 失敗: ${res.status} ${text}`);
      }

      // 期待レスポンス: MapResponse { id, name, imageData(=plain Base64), ... }
      const data = await res.json();

      // API優先。なければ今回のpayload(=新規アップロード) or 初期(dataURL→plain)を採用
      const nextImageDataPlain: string | null =
        data.imageData ?? (payload.imageData ?? dataUrlToPlainBase64(initialImageUrl));

      onSaved?.({
        id: data.id ?? mapId,
        name: data.name ?? mapName,
        imageData: nextImageDataPlain ?? null,
      });
    } catch (e) {
      console.error(e);
      alert("マップの保存に失敗しました。ログを確認してください。");
    } finally {
      setSaving(false);
    }
  };

  const canSave = !!mapId && !saving;

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title"> マップを編集</h2>

      <MapNameForm value={mapName} onChange={setMapName} />

      <label className="Label">マップ画像</label>
      <div className="fileinputContainer">
        <MapImageUploadForm value={mapImage} onChange={setMapImage} />
        {previewUrl && (
          <div>
            <PreviewMapImage image={previewUrl} />
          </div>
        )}
      </div>

      <div className="buttonRow">
        <SaveButton onClick={handleSave} disabled={!canSave} />
        <DeleteButton />
      </div>
    </div>
  );
};
