// src/components/admin/SideMenu/MapEditForm/MapEditForm.tsx
"use client";
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
    imageData?: string | null;           // API仕様に合わせて“ヘッダ無しBase64”で返す
    // 必要ならここに naturalWidth/naturalHeight を足してもOK
  }) => void;

  // 削除完了後に親マップへ遷移させるためのコールバック
  // 引数: 遷移先の parentMapId
  onDeleted?: (parentMapId: string) => void;
};

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

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
  onDeleted,
}) => {
  const [mapName, setMapName] = useState(initialName);
  const [mapImage, setMapImage] = useState<File | null>(null);
  const [saving, setSaving] = useState(false);

  // 計測した自然サイズ（previewUrl から都度測定）
  const [naturalSize, setNaturalSize] = useState<{ w: number; h: number }>({ w: 0, h: 0 });

  // 親マップID（= 削除可否と削除後の遷移に利用）
  const [parentMapId, setParentMapId] = useState<string | null>(null);
  const [loadingMapMeta, setLoadingMapMeta] = useState(false);

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

  // previewUrl が変わるたびに naturalWidth/Height を測定
  useEffect(() => {
    if (!previewUrl) {
      setNaturalSize({ w: 0, h: 0 });
      return;
    }
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => {
      setNaturalSize({ w: img.naturalWidth || 0, h: img.naturalHeight || 0 });
    };
    img.onerror = () => {
      setNaturalSize({ w: 0, h: 0 });
    };
    img.src = previewUrl;
  }, [previewUrl]);

  // マップ詳細を取得して parentMapId を保持（削除可否に利用）
  useEffect(() => {
    if (!mapId) {
      setParentMapId(null);
      return;
    }
    let abort = false;
    (async () => {
      try {
        setLoadingMapMeta(true);
        const res = await fetch(`${API}/maps/${encodeURIComponent(mapId)}`, {
          credentials: "include",
        });
        if (!res.ok) {
          console.warn("GET /maps/{mapId} 失敗:", res.status);
          if (!abort) setParentMapId(null);
          return;
        }
        const data = await res.json();
        if (!abort) setParentMapId(data?.parentMapId ?? null);
        if (!abort && !initialName && data?.name) {
          setMapName(String(data.name));
        }
      } catch (e) {
        console.error(e);
        if (!abort) setParentMapId(null);
      } finally {
        if (!abort) setLoadingMapMeta(false);
      }
    })();
    return () => {
      abort = true;
    };
  }, [mapId]); // initialName は依存にしない

  const handleSave = async () => {
    // バリデーション
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

      // PATCH /maps/{mapId} の部分更新（必要な項目のみ送る）
      const payload: Record<string, any> = {};

      // name の変更
      if (mapName && mapName !== initialName) payload.name = mapName;

      // 画像の変更（Base64 + 自然サイズをセット）
      if (mapImage) {
        payload.imageData = await fileToBase64Plain(mapImage); // Base64(ヘッダ無し)
        if (naturalSize.w > 0 && naturalSize.h > 0) {
          payload.naturalWidth = Math.round(naturalSize.w);
          payload.naturalHeight = Math.round(naturalSize.h);
        }
      }

      // 変更がなければ即時成功扱い
      if (Object.keys(payload).length === 0) {
        onSaved?.({
          id: mapId,
          name: mapName,
          imageData: dataUrlToPlainBase64(initialImageUrl),
        });
        return;
      }

      const res = await fetch(`${API}/maps/${encodeURIComponent(mapId)}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const text = await res.text();
        throw new Error(`PATCH /maps/${mapId} 失敗: ${res.status} ${text}`);
      }

      const data = await res.json();

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

  // 削除（親がある場合のみ）。親が Index の場合は差し替え手順を実行。
  const handleDelete = async () => {
    if (!parentMapId) {
      // ★ ここで Index Map へのアラートを明示的に表示
      alert("Index Mapは削除できません！");
      return;
    }
    if (!confirm("このマップを削除しますか？この操作は取り消せません。")) return;

    try {
      // 親マップの詳細を取得して、親が Index（= さらに親が無い）か判定
      const parentRes = await fetch(`${API}/maps/${encodeURIComponent(parentMapId)}`, {
        credentials: "include",
      });
      if (!parentRes.ok) {
        const text = await parentRes.text();
        throw new Error(`GET /maps/${parentMapId} 失敗: ${parentRes.status} ${text}`);
      }
      const parent = await parentRes.json();
      const parentIsIndex = !parent?.parentMapId;

      if (parentIsIndex) {
        // 1) 新しい空マップを作成（親を明示的に紐付け）
        const createRes = await fetch(`${API}/maps`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify({ name: "untitled map", parentMapId: parentMapId }),
        });
        if (!createRes.ok) {
          const text = await createRes.text();
          throw new Error(`POST /maps 失敗: ${createRes.status} ${text}`);
        }
        const newMap = await createRes.json();
        const newMapId: string = newMap?.id;

        // 2) 親の pins を取得し、linkToMapId === 現在の mapId のエリアピンを探す
        const pinsRes = await fetch(`${API}/maps/${encodeURIComponent(parentMapId)}/pins`, {
          credentials: "include",
        });
        if (!pinsRes.ok) {
          const text = await pinsRes.text();
          throw new Error(`GET /maps/${parentMapId}/pins 失敗: ${pinsRes.status} ${text}`);
        }
        const pins: any[] = await pinsRes.json();
        const linkPin = pins.find(
          (p) => p?.type === "area_selector" && p?.linkToMapId === mapId
        );

        // 2') ピンの linkToMapId を新マップに付け替え
        if (linkPin?.id && newMapId) {
          const patchRes = await fetch(`${API}/pins/${encodeURIComponent(linkPin.id)}`, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({ linkToMapId: newMapId }),
          });
          if (!patchRes.ok) {
            const text = await patchRes.text();
            throw new Error(`PATCH /pins/${linkPin.id} 失敗: ${patchRes.status} ${text}`);
          }
        } else {
          console.warn("親マップ上に対象リンクピンが見つかりませんでした。");
        }

        // 3) 対象マップを削除
        const delRes = await fetch(`${API}/maps/${encodeURIComponent(mapId)}`, {
          method: "DELETE",
          credentials: "include",
        });
        if (!delRes.ok) {
          const text = await delRes.text();
          throw new Error(`DELETE /maps/${mapId} 失敗: ${delRes.status} ${text}`);
        }

        // 親マップへ遷移（呼び出し元に委譲）
        onDeleted?.(parentMapId);
        onClose();
        return;
      }

      // 親が Index でない場合：単純に削除して親へ戻る
      const delRes = await fetch(`${API}/maps/${encodeURIComponent(mapId)}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!delRes.ok) {
        const text = await delRes.text();
        throw new Error(`DELETE /maps/${mapId} 失敗: ${delRes.status} ${text}`);
      }
      onDeleted?.(parentMapId);
      onClose();
    } catch (e) {
      console.error(e);
      alert("マップの削除に失敗しました。ログを確認してください。");
    }
  };

  const canSave = !!mapId && !saving;
  const disableDeleteUI = loadingMapMeta || !parentMapId; // 親が無ければ削除不可（Index）

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
          <DeleteButton onClick={handleDelete} />
        </div>

      {!parentMapId && (
        <p className="mt-2 text-xs text-gray-600">
          親マップが存在しないマップ（ルート / Index Map）は削除できません。
        </p>
      )}
    </div>
  );
};
