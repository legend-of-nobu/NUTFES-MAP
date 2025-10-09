"use client";
import React, { useEffect, useMemo, useState } from "react";
import PlanNameForm from "./PlanNameForm";
import PlanCategoryForm from "./PlanCategoryForm";
import WaitTimeForm from "./WaitTimeForm";
import PlanImageForm from "./PlanImageForm/PlanImageForm";
import PlanExplainForm from "./PlanExplainForm";
import PlanPlaceForm from "./PlanPlaceForm";
import PlanClosedForm from "./PlanClosedForm";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import type { ApiPin } from "@/components/map/PlanPin";
import "../FormStyle.css";
import "../Style.css";

/** 既存: 新規作成時のコンテキスト */
type CreateCtx = {
  mapId: string | null;
  draftPos: { xNorm: number; yNorm: number } | null;
  onCreated?: (p: ApiPin) => void;
};

/** 追加: 既存編集時のコンテキスト */
type EditCtx = {
  mapId: string;            // 対象の親マップID（UI側で使うために保持）
  pin: ApiPin;              // 既存ピン
  onUpdated?: (p: ApiPin) => void;
  onDeleted?: (pinId: string) => void;
};

export const PlanEditForm: React.FC<{
  onClose: () => void;
  context?: CreateCtx;  // 新規
  edit?: EditCtx;       // 既存編集
}> = ({ onClose, context, edit }) => {
  const isEdit = !!edit;

  const [planName, setPlanName] = useState("");
  const [category, setCategory] = useState(""); // "food" | "child" | "plan"
  const [waitTime, setWaitTime] = useState(""); // number string
  const [place, setPlace] = useState("");
  const [image, setImage] = useState<string | null>(null); // dataURL or plain base64
  const [description, setDescription] = useState("");
  const [closed, setClosed] = useState(false);
  const [saving, setSaving] = useState(false);

  const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

  // 既存編集時は初期値を既存ピンから投入
  useEffect(() => {
    if (!isEdit || !edit?.pin) return;
    const p = edit.pin;
    setPlanName(p.name ?? "");
    setCategory(p.category ?? "plan");
    setWaitTime(String(p.waitMinutes ?? 0));
    setPlace(p.place ?? "");
    setImage(p.descriptionImageData ?? null); // dataURLでもplainでもOK
    setDescription(p.description ?? "");
    setClosed((p.status ?? "open") === "closed");
  }, [isEdit, edit?.pin?.id]); // 別ピンへ切替時も反映

  // 保存可否
  const canSave = useMemo(() => {
    if (saving) return false;
    if (isEdit) {
      return !!edit?.mapId && !!edit?.pin?.id && !!planName.trim() && !!category.trim();
    }
    // 新規
    return !!context?.mapId && !!context?.draftPos && !!planName.trim() && !!category.trim();
  }, [saving, isEdit, edit?.mapId, edit?.pin?.id, context?.mapId, context?.draftPos, planName, category]);

  // dataURL → plain Base64（既に plain の場合はそのまま）
  const toPlainBase64 = (src: string | null) => {
    if (!src) return null;
    if (src.startsWith("data:")) return src.replace(/^data:.*;base64,/, "");
    return src;
  };

  // 保存（新規: POST /maps/{mapId}/pins, 既存: PATCH /pins/{pinId}）
  const handleSave = async () => {
    if (!canSave) return;

    try {
      setSaving(true);

      const trimmedPlace = place.trim();
      const payload = {
        name: planName.trim(),
        description: description.trim() || undefined,
        descriptionImageData: toPlainBase64(image) || undefined,
        // 既存編集では位置や種別は変更しない（要件により拡張可）
        category,
        status: closed ? "closed" : "open",
        waitMinutes: Number.isFinite(Number(waitTime)) ? Number(waitTime) : 0,
        place: trimmedPlace ? trimmedPlace : isEdit ? null : undefined,
      };

      if (isEdit) {
        const { pin } = edit!;
        // ★ 修正点: PATCH は /pins/{pinId}
        const res = await fetch(`${API}/pins/${encodeURIComponent(pin.id)}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(payload),
        });
        if (!res.ok) {
          const text = await res.text();
          throw new Error(`企画ピンの更新に失敗しました: ${res.status} ${text}`);
        }
        const p = await res.json();
        const updated: ApiPin = {
          id: p.id ?? pin.id,
          mapId: p.mapId ?? pin.mapId,
          name: p.name ?? planName,
          description: p.description ?? description || null,
          descriptionImageData: p.descriptionImageData ?? (toPlainBase64(image) || null),
          type: p.type ?? pin.type,
          linkToMapId: p.linkToMapId ?? pin.linkToMapId ?? null,
          xNorm: p.xNorm ?? pin.xNorm,
          yNorm: p.yNorm ?? pin.yNorm,
          place: p.place ?? (trimmedPlace ? trimmedPlace : null),
          category: p.category ?? category,
          status: p.status ?? (closed ? "closed" : "open"),
          waitMinutes: p.waitMinutes ?? Number(waitTime) || 0,
          createdAt: p.createdAt ?? pin.createdAt ?? new Date().toISOString(),
          modifiedAt: p.modifiedAt ?? new Date().toISOString(),
        };
        edit?.onUpdated?.(updated);
        onClose();
        return;
      }

      // === 新規作成 ===
      const { mapId, draftPos } = context!;
      const createPayload = {
        ...payload,
        type: "exhibit",
        // plan は通常リンク不要（必要なら UI 側で別用途に使用）
        // linkToMapId: undefined,
        xNorm: draftPos!.xNorm,
        yNorm: draftPos!.yNorm,
      };

      const res = await fetch(`${API}/maps/${encodeURIComponent(mapId!)}/pins`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(createPayload),
      });

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
        place: p.place ?? (trimmedPlace ? trimmedPlace : null),
        category: p.category ?? "plan",
        status: p.status ?? "open",
        waitMinutes: p.waitMinutes ?? 0,
        createdAt: p.createdAt ?? new Date().toISOString(),
        modifiedAt: p.modifiedAt ?? new Date().toISOString(),
      };
      context?.onCreated?.(created);
      onClose();
    } catch (err) {
      console.error(err);
      alert(isEdit ? "企画ピンの更新に失敗しました。ログを確認してください。" : "企画ピンの保存に失敗しました。ログを確認してください。");
    } finally {
      setSaving(false);
    }
  };

  // 削除（既存編集時のみ有効）: DELETE /pins/{pinId}
  const handleDelete = async () => {
    if (!isEdit || !edit) return;
    if (!confirm("この企画ピンを削除しますか？")) return;

    try {
      setSaving(true);
      // ★ 修正点: DELETE も /pins/{pinId}
      const res = await fetch(`${API}/pins/${encodeURIComponent(edit.pin.id)}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(`企画ピンの削除に失敗しました: ${res.status} ${text}`);
      }
      edit.onDeleted?.(edit.pin.id);
      onClose();
    } catch (err) {
      console.error(err);
      alert("企画ピンの削除に失敗しました。ログを確認してください。");
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title">{isEdit ? "ピンを編集" : "ピンを作成"}</h2>

      <PlanNameForm value={planName} onChange={setPlanName} />
      <PlanCategoryForm value={category} onChange={setCategory} />
      <WaitTimeForm value={waitTime} onChange={setWaitTime} />
      <PlanPlaceForm value={place} onChange={setPlace} />
      <PlanClosedForm value={closed} onChange={setClosed} />
      <PlanImageForm value={image} onChange={setImage} />
      <PlanExplainForm value={description} onChange={setDescription} />

      <div className="buttonRow">
        <SaveButton onClick={handleSave} disabled={!canSave || saving} />
        <DeleteButton onClick={isEdit ? handleDelete : undefined} disabled={!isEdit || saving} />
      </div>

      {!isEdit && !context?.draftPos && (
        <p className="mt-2 text-xs text-gray-600">
          「決定」で設置モードに入り、マップ上をクリックして位置を確定してから保存してください。
        </p>
      )}
    </div>
  );
};
