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
  const [mapEditTarget, setMapEditTarget] = useState<MapType | null>(null);

  // ★ 追加: 既存エリアピン編集対象
  const [areaEditTarget, setAreaEditTarget] = useState<ApiAreaPin | null>(null);

  const openMapEdit = () => {
    if (!selectedMap) {
      alert("編集するマップを先に選択してください。");
      return;
    }
    setMapEditTarget(selectedMap);
    setSideMenuMode("map");
  };
  const closeSideMenu = () => {
    setSideMenuMode(null);
    setMapEditTarget(null);
    setAreaEditTarget(null);
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
          Array.isArray((data as any)?.items) ? (data as any).items :
          Array.isArray(data) ? (data as any) :
          data ? [data as any] : [];
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
    if (draftPos) return;
    if (!placingKind || !selectedMap) return;
    setDraftPos({ xNorm, yNorm });
    setSideMenuMode(placingKind); // "area" or "plan"
  };

  // PlanPin クリック → BottomSheet
  const handlePlanPinSelect = (spot: SpotData) => {
    setSelectedSpot(spot);
    setIsSheetOpen(true);
  };

  // 単一マップ取得
  const fetchMapById = async (mapId: string): Promise<MapType | null> => {
    try {
      const res = await fetch(`${API}/maps/${mapId}`, { credentials: "include" });
      if (!res.ok) return null;
      const m = await res.json();
      return {
        id: m.id,
        name: m.name ?? "",
        imageData: m.imageData ?? null,
        naturalWidth: m.naturalWidth ?? 0,
        naturalHeight: m.naturalHeight ?? 0,
        parentMapId: m.parentMapId ?? null,
      };
    } catch {
      return null;
    }
  };

  const goToMapId = async (mapId: string) => {
    const existing = maps.find((m) => m.id === mapId);
    if (existing) {
      setSelectedMap(existing);
      return;
    }
    const mapObj = await fetchMapById(mapId);
    if (mapObj) {
      setMaps((prev) => (prev.some((x) => x.id === mapObj.id) ? prev : [...prev, mapObj]));
      setSelectedMap(mapObj);
    } else {
      setSelectedMap({ id: mapId, name: "", parentMapId: null });
    }
  };

  // ★ 修正: エリアピンをクリック → Editモードなら Area 編集サイドバー、Viewモードなら遷移
  const handleAreaPinSelect = async (area: ApiAreaPin) => {
    if (mode === "edit") {
      setAreaEditTarget(area);      // 既存ピンを編集対象に
      setSideMenuMode("area");      // AreaEdit を開く
      return;
    }
    // userモードは従来どおりリンク先へ遷移
    if (area.linkToMapId) {
      await goToMapId(area.linkToMapId);
    }
  };

  const handleBackToParent = async () => {
    const parentId = selectedMap?.parentMapId;
    if (!parentId) return;
    await goToMapId(parentId);
  };

  // 既存エリアピンの名称更新を反映
  const applyAreaPinUpdated = (p: ApiAreaPin) =>
    setAreaPins((prev) => prev.map((ap) => (ap.id === p.id ? p : ap)));

  // 新規作成のコールバック（既存機能は維持）
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
          onAddPinAt={placingKind ? handleAddPinAt : undefined}
          placing={!!placingKind}
          placingKind={placingKind}
          draftPos={draftPos}
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
            sideMenuMode === "map" && mapEditTarget
              ? {
                  mapId: mapEditTarget.id,
                  initialName: mapEditTarget.name,
                  initialImageUrl: toPreviewUrl(mapEditTarget.imageData ?? null),
                  onSaved: async (updated) => {
                    const fresh = await fetchMapById(updated.id);
                    if (fresh) {
                      setMaps((prev) =>
                        prev.some((m) => m.id === fresh.id)
                          ? prev.map((m) => (m.id === fresh.id ? { ...m, ...fresh } : m))
                          : [...prev, fresh]
                      );
                      setSelectedMap((prev) =>
                        prev && prev.id === fresh.id ? { ...prev, ...fresh } : prev
                      );
                      setMapEditTarget((prev) =>
                        prev && prev.id === fresh.id ? { ...prev, ...fresh } : prev
                      );
                    } else {
                      setMaps((prev) =>
                        prev.map((m) =>
                          m.id === updated.id
                            ? { ...m, name: updated.name, imageData: updated.imageData ?? m.imageData }
                            : m
                        )
                      );
                      if (selectedMap?.id === updated.id) {
                        setSelectedMap({
                          ...selectedMap,
                          name: updated.name,
                          imageData: updated.imageData ?? selectedMap.imageData,
                        });
                      }
                      if (mapEditTarget?.id === updated.id) {
                        setMapEditTarget({
                          ...mapEditTarget,
                          name: updated.name,
                          imageData: updated.imageData ?? mapEditTarget.imageData,
                        });
                      }
                    }
                    closeSideMenu();
                  },
                }
              : undefined
          }
          // ★ エリア編集（既存ピンの名称変更）
          areaEditTarget={sideMenuMode === "area" ? areaEditTarget : null}
          onAreaUpdated={applyAreaPinUpdated}
          // ★ 既存機能（新規設置用コンテキスト）は維持
          pinContext={{
            placingKind: placingKind,
            mapId: selectedMap?.id ?? null,
            draftPos: draftPos,
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
