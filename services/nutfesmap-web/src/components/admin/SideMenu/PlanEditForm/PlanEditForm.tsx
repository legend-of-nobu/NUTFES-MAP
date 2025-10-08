"use client";
import React, { useMemo, useState } from "react";
import PlanNameForm from "./PlanNameForm";
import PlanCategoryForm from "./PlanCategoryForm";
import WaitTimeForm from "./WaitTimeForm";
import PlanImageForm from "./PlanImageForm/PlanImageForm";
import PlanExplainForm from "./PlanExplainForm";
import PlanClosedForm from "./PlanClosedForm";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import type { ApiPin } from "@/components/map/PlanPin";
import "../FormStyle.css";
import "../Style.css";

/**
 * AdminPage → SideMenu から渡ってくる新規作成用コンテキスト
 * - mapId: 親マップ
 * - draftPos: MapImage 内で確定した正規化座標（ゴースト/設置モードは既存のまま）
 * - onCreated: 作成後に親へ通知して画面のピン配列へ即時反映
 */
type CreateCtx = {
  mapId: string | null;
  draftPos: { xNorm: number; yNorm: number } | null;
  onCreated?: (p: ApiPin) => void;
};

export const PlanEditForm: React.FC<{ onClose: () => void; context?: CreateCtx }> = ({
  onClose,
  context,
}) => {
  const [planName, setPlanName] = useState("");
  const [category, setCategory] = useState(""); // "food" | "child" | "plan"
  const [waitTime, setWaitTime] = useState(""); // number string
  const [image, setImage] = useState<string | null>(null); // dataURL or plain base64
  const [description, setDescription] = useState("");
  const [closed, setClosed] = useState(false);
  const [saving, setSaving] = useState(false);

  const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

  // 保存可否（ゴースト・設置フローはそのまま。mapId と draftPos が無いと保存できない）
  const canSave = useMemo(
    () =>
      !!context?.mapId &&
      !!context?.draftPos &&
      !!planName.trim() &&
      !!category.trim() &&
      !saving,
    [context?.mapId, context?.draftPos, planName, category, saving]
  );

  // dataURL → plain Base64（既に plain の場合はそのまま）
  const toPlainBase64 = (src: string | null) => {
    if (!src) return null;
    if (src.startsWith("data:")) return src.replace(/^data:.*;base64,/, "");
    return src;
    // NOTE: API 側は descriptionImageData を “ヘッダ無し Base64” で受けます
  };

  const handleSave = async () => {
    if (!canSave) return;

    try {
      setSaving(true);

      const { mapId, draftPos } = context!;
      const payload = {
        name: planName.trim(),
        description: description.trim() || undefined,
        descriptionImageData: toPlainBase64(image) || undefined,
        type: "exhibit", // 企画ピン
        linkToMapId: null,
        xNorm: draftPos!.xNorm,
        yNorm: draftPos!.yNorm,
        category, // backend: "food" | "child" | "plan"
        status: closed ? "closed" : "open",
        waitMinutes: Number.isFinite(Number(waitTime)) ? Number(waitTime) : 0,
      };

      // POST /maps/{mapId}/pins（※複数形 /pins）
      const res = await fetch(
        `${API}/maps/${encodeURIComponent(mapId!)}/pins`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(payload),
        }
      );

      if (!res.ok) {
        const text = await res.text();
        throw new Error(`企画ピンの作成に失敗しました: ${res.status} ${text}`);
      }

      const p = await res.json();

      const created: ApiPin = {
        id: p.id,
        mapId: p.mapId,
        name: p.name,
        description: p.description ?? null,
        descriptionImageData: p.descriptionImageData ?? null,
        type: p.type,
        linkToMapId: p.linkToMapId ?? null,
        xNorm: p.xNorm,
        yNorm: p.yNorm,
        category: p.category ?? "plan",
        status: p.status ?? "open",
        waitMinutes: p.waitMinutes ?? 0,
        createdAt: p.createdAt ?? new Date().toISOString(),
        modifiedAt: p.modifiedAt ?? new Date().toISOString(),
      };

      // 親（AdminPage）へ通知してピン配列に追加（マップは既存のまま。ゴースト等の挙動も不変）
      context?.onCreated?.(created);

      onClose();
    } catch (err) {
      console.error(err);
      alert("企画ピンの保存に失敗しました。ログを確認してください。");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title"> ピンを編集</h2>

      <PlanNameForm value={planName} onChange={setPlanName} />
      <PlanCategoryForm value={category} onChange={setCategory} />
      <WaitTimeForm value={waitTime} onChange={setWaitTime} />
      <PlanClosedForm value={closed} onChange={setClosed} />
      <PlanImageForm value={image} onChange={setImage} />
      <PlanExplainForm value={description} onChange={setDescription} />

      <div className="buttonRow">
        <SaveButton onClick={handleSave} disabled={!canSave} />
        {/* 新規作成画面では削除は無効化（編集は別フローで対応） */}
        <DeleteButton disabled />
      </div>

      {!context?.draftPos && (
        <p className="mt-2 text-xs text-gray-600">
          「決定」で設置モードに入り、マップ上をクリックして位置を確定してから保存してください。
        </p>
      )}
    </div>
  );
};
