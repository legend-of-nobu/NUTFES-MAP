"use client";
import { useEffect, useMemo, useState } from "react";
import AdminHeader from "./AdminHeader";
import MapHeader from "./MapHeader";
import Map from "./Map";
import PinKindSelectModal from "./PinKindSelectModal";
import SideMenu from "./SideMenu/SideMenu";
import { toPreviewUrl } from "./SideMenu/MapEditForm/base64";
import PlanSpotBottomSheet from "@/components/map/PlanSpotBottomSheet";

import type { ApiPin, SpotData } from "@/components/map/PlanPin";
import type { ApiAreaPin } from "@/components/map/AreaPin";

type MapType = {
  id: string;
  name: string;
  imageData?: string | null;
  naturalWidth?: number;
  naturalHeight?: number;
  parentMapId?: string | null;
};

type PinKind = "area" | "plan";

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

export default function AdminPage() {
  const [maps, setMaps] = useState<MapType[]>([]);
  const [selectedMap, setSelectedMap] = useState<MapType | null>(null);

  // 実ピン（表示用）
  const [planPins, setPlanPins] = useState<ApiPin[]>([]);
  const [areaPins, setAreaPins] = useState<ApiAreaPin[]>([]);

  const [mode, setMode] = useState<"edit" | "user">("edit");

  // 右側サイドメニュー
  const [sideMenuMode, setSideMenuMode] = useState<"map" | "plan" | "area" | null>(null);
  const openMapEdit = () => {
    if (!selectedMap) {
      alert("編集するマップを先に選択してください。");
      return;
    }
    setSideMenuMode("map");
  };
  const closeSideMenu = () => {
    setSideMenuMode(null);
    // 設置モード終了（ゴースト除去）
    setPlacingKind(null);
    setDraftPos(null);
  };

  // 「＋ピン」モーダル
  const [showPinKindModal, setShowPinKindModal] = useState(false);
  const [selectedKind, setSelectedKind] = useState<PinKind | null>(null);

  // 設置モード：選択中のピン種別と確定座標
  const [placingKind, setPlacingKind] = useState<PinKind | null>(null);
  const [draftPos, setDraftPos] = useState<{ xNorm: number; yNorm: number } | null>(null);

  // 企画ピン BottomSheet（閲覧用）
  const [isSheetOpen, setIsSheetOpen] = useState(false);
  const [selectedSpot, setSelectedSpot] = useState<SpotData | null>(null);

  // === 初期ロード：マップ一覧 ===
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API}/maps/index`, { credentials: "include" });
        const data = await res.json();
        const items: MapType[] =
          Array.isArray(data?.items) ? data.items :
          Array.isArray(data) ? data :
          data ? [data] : [];
        setMaps(items);
        setSelectedMap((prev) => prev ?? (items.length > 0 ? items[0] : null));
      } catch (e) {
        console.error("Failed to fetch maps:", e);
      }
    })();
  }, []);

  // === ピン一覧取得（選択マップ変更時） ===
  useEffect(() => {
    if (!selectedMap?.id) return;
    (async () => {
      try {
        const res = await fetch(`${API}/maps/${selectedMap.id}/pins`, {
          credentials: "include",
        });
        if (!res.ok) {
          console.warn("Failed to fetch pins:", res.status);
          setPlanPins([]);
          setAreaPins([]);
          return;
        }
        const pins = (await res.json()) as any[];

        const plans: ApiPin[] = [];
        const areas: ApiAreaPin[] = [];
        for (const p of pins) {
          if (p.type === "area_selector") {
            areas.push({
              id: p.id,
              mapId: p.mapId,
              name: p.name,
              xNorm: p.xNorm,
              yNorm: p.yNorm,
              linkToMapId: p.linkToMapId ?? null,
            });
          } else {
            plans.push({
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
            });
          }
        }
        setPlanPins(plans);
        setAreaPins(areas);
      } catch (e) {
        console.error("Failed to fetch pins:", e);
        setPlanPins([]);
        setAreaPins([]);
      }
    })();
  }, [selectedMap?.id]);

  // 「＋ピン」クリック → モーダル
  const handleOpenPinKindModal = () => {
    setSelectedKind(null);
    setShowPinKindModal(true);
  };
  const handleConfirmPinKind = () => {
    if (!selectedKind) return;
    // 設置開始（追従モード）
    setPlacingKind(selectedKind);
    setDraftPos(null);
    setShowPinKindModal(false);
  };

  // MapImage 内クリックで座標確定 → サイドメニュー起動
  const handleAddPinAt = (xNorm: number, yNorm: number) => {
    // 既に確定済みなら無視（ゴーストは固定のまま）
    if (draftPos) return;
    if (!placingKind || !selectedMap) return;
    setDraftPos({ xNorm, yNorm });   // ← 以降は追従停止してゴースト固定
    setSideMenuMode(placingKind);    // "area" or "plan" を開く
  };

  // PlanPin クリック → BottomSheet
  const handlePlanPinSelect = (spot: SpotData) => {
    setSelectedSpot(spot);
    setIsSheetOpen(true);
  };

  // AreaPin クリック → linkToMapId のマップへ遷移
  const handleAreaPinSelect = async (area: ApiAreaPin) => {
    if (!area.linkToMapId) return;
    await goToMapId(area.linkToMapId);
  };

  const handleBackToParent = async () => {
    const parentId = selectedMap?.parentMapId;
    if (!parentId) return;
    await goToMapId(parentId);
  };

  const goToMapId = async (mapId: string) => {
    const existing = maps.find((m) => m.id === mapId);
    if (existing) {
      setSelectedMap(existing);
      return;
    }
    try {
      const res = await fetch(`${API}/maps/${mapId}`, { credentials: "include" });
      if (!res.ok) {
        setSelectedMap({ id: mapId, name: "", parentMapId: null });
        return;
      }
      const m = await res.json();
      const mapObj: MapType = {
        id: m.id,
        name: m.name ?? "",
        imageData: m.imageData ?? null,
        naturalWidth: m.naturalWidth ?? 0,
        naturalHeight: m.naturalHeight ?? 0,
        parentMapId: m.parentMapId ?? null,
      };
      setMaps((prev) => (prev.some((x) => x.id === mapObj.id) ? prev : [...prev, mapObj]));
      setSelectedMap(mapObj);
    } catch {
      setSelectedMap({ id: mapId, name: "", parentMapId: null });
    }
  };

  // サイドメニュー経由で Area/Plan 作成完了時に配列追加
  const appendAreaPin = (p: ApiAreaPin) => setAreaPins((prev) => [...prev, p]);
  const appendPlanPin = (p: ApiPin) => setPlanPins((prev) => [...prev, p]);

  const headerNode = useMemo(
    () => (
      <AdminHeader
        currentMapName={selectedMap?.name ?? "未選択"}
        mode={mode}
        onModeChange={(m) => setMode(m)}
        onMapEdit={openMapEdit}
        onStairAdd={() => console.log("階を追加")}
        onStairDelete={() => console.log("階を削除")}
      />
    ),
    [selectedMap?.name, mode]
  );

  return (
    <div className="flex flex-col h-screen bg-[#f5f0dc]">
      <MapHeader
        mapName={selectedMap?.name ?? ""}
        parentMapId={selectedMap?.parentMapId ?? null}
        onBack={handleBackToParent}
      />

      {/* Map 本体 */}
      <div className="relative flex-1 min-h-0 flex">
        <Map
          pins={planPins}
          areaPins={areaPins}
          mode={mode}
          onPlanPinSelect={handlePlanPinSelect}
          onAreaPinSelect={handleAreaPinSelect}
          onAddPin={mode === "edit" ? handleOpenPinKindModal : undefined}
          // 設置モード：クリック確定で draftPos をセット（以降追従停止）
          onAddPinAt={placingKind ? handleAddPinAt : undefined}
          placing={!!placingKind}
          placingKind={placingKind}
          draftPos={draftPos} // ★ 追加（ゴースト固定用）
          mapId={selectedMap?.id ?? null}
          mapImageData={toPreviewUrl(selectedMap?.imageData ?? null)}
          naturalWidth={selectedMap?.naturalWidth ?? 4096}
          naturalHeight={selectedMap?.naturalHeight ?? 3072}
          header={headerNode}
        />
      </div>

      {/* SideMenu（Map編集/Plan編集/Area編集） */}
      {sideMenuMode && (
        <SideMenu
          mode={sideMenuMode}
          onClose={closeSideMenu}
          // map 編集
          mapEditProps={
            sideMenuMode === "map" && selectedMap
              ? {
                  mapId: selectedMap.id,
                  initialName: selectedMap.name,
                  initialImageUrl: toPreviewUrl(selectedMap.imageData ?? null),
                  onSaved: (updated) => {
                    setMaps((prev) =>
                      prev.map((m) =>
                        m.id === updated.id
                          ? {
                              ...m,
                              name: updated.name,
                              imageData: updated.imageData ?? m.imageData,
                            }
                          : m
                      )
                    );
                    setSelectedMap((prev) =>
                      prev && prev.id === updated.id
                        ? {
                            ...prev,
                            name: updated.name,
                            imageData: updated.imageData ?? prev.imageData,
                          }
                        : prev
                    );
                    closeSideMenu();
                  },
                }
              : undefined
          }
          // ★ ピン設置用コンテキスト（保存で実ピン追加 → closeSideMenu でゴースト除去）
          pinContext={{
            placingKind: placingKind,
            mapId: selectedMap?.id ?? null,
            draftPos: draftPos, // {xNorm,yNorm}
            onAreaCreated: appendAreaPin,
            onPlanCreated: appendPlanPin,
          }}
        />
      )}

      {/* 「＋ピン」種別選択モーダル */}
      <PinKindSelectModal
        visible={showPinKindModal}
        selected={selectedKind}
        onSelect={(kind) => setSelectedKind(kind as PinKind)}
        onConfirm={handleConfirmPinKind}
        onCancel={() => {
          setShowPinKindModal(false);
          setSelectedKind(null);
        }}
      />

      {/* 企画ピン詳細 BottomSheet */}
      <PlanSpotBottomSheet
        isOpen={isSheetOpen}
        spotData={selectedSpot}
        onClose={() => setIsSheetOpen(false)}
      />
    </div>
  );
}
