import React, { useState } from "react";
import { MapNameForm } from "./MapNameForm";
import { MapImageUploadForm } from "./MapImageUploadForm";
import PreviewMapImage from "./PreviewMapImage";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import "../Style.css";
import "../Image.css";

export type MapEditFormProps = {
  onClose: () => void;
  mapId: string;                         // 編集対象のMap ID（必須）
  initialName?: string;                  // 初期表示用マップ名
  initialImageUrl?: string | null;       // 初期プレビュー用（dataURL or 署名URL）
  onSaved?: (updated: {
    id: string;
    name: string;
    imageData?: string | null;
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

  const previewUrl = mapImage ? URL.createObjectURL(mapImage) : initialImageUrl ?? null;

  const handleSave = async () => {
    if (!mapId) {
      alert("編集対象のマップが選択されていません。");
      return;
    }
    try {
      setSaving(true);

      // OAS: PATCH /maps/{mapId} の部分更新（必要な項目のみ送る）
      const payload: Record<string, any> = {};
      if (mapName && mapName !== initialName) payload.name = mapName;
      if (mapImage) payload.imageData = await fileToBase64Plain(mapImage); // format: byte（Base64）

      // 変更がなければ即時成功扱い
      if (Object.keys(payload).length === 0) {
        onSaved?.({ id: mapId, name: mapName, imageData: initialImageUrl ?? null });
        return;
      }

      const res = await fetch(
        `${process.env.NEXT_PUBLIC_API_BASE_URL}/maps/${encodeURIComponent(mapId)}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            // 必要に応じて CSRF トークンや Authorization を追加
            // "X-CSRF-Token": csrfToken,
          },
          credentials: "include", // Cookie 認証想定
          body: JSON.stringify(payload),
        }
      );

      if (!res.ok) {
        const text = await res.text();
        throw new Error(`PATCH /maps/${mapId} 失敗: ${res.status} ${text}`);
      }

      // 期待レスポンス: MapResponse { id, name, imageData, ... }
      const data = await res.json();

      onSaved?.({
        id: data.id ?? mapId,
        name: data.name ?? mapName,
        imageData: data.imageData ?? initialImageUrl ?? null,
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
