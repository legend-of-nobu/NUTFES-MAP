"use client";
import { useCallback, useEffect, useMemo, useState } from "react";
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
  hasFloors?: boolean;
  floorCount?: number;
};

type MapPatch = Partial<Omit<MapType, "id">> & { id: string };

const mapShallowEqual = (a: MapType, b: MapType) =>
  a.id === b.id &&
  a.name === b.name &&
  (a.imageData ?? null) === (b.imageData ?? null) &&
  (a.naturalWidth ?? 0) === (b.naturalWidth ?? 0) &&
  (a.naturalHeight ?? 0) === (b.naturalHeight ?? 0) &&
  (a.parentMapId ?? null) === (b.parentMapId ?? null) &&
  (a.hasFloors ?? false) === (b.hasFloors ?? false) &&
  (a.floorCount ?? 0) === (b.floorCount ?? 0);

const mergeMap = (original: MapType, patch: MapPatch): MapType => {
  const merged: MapType = { ...original, ...patch };
  return mapShallowEqual(original, merged) ? original : merged;
};

type PinKind = "area" | "plan";
const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

export default function AdminPage() {
  const [maps, setMaps] = useState<MapType[]>([]);
  const [selectedMap, setSelectedMap] = useState<MapType | null>(null);
  const [floorItems, setFloorItems] = useState<Array<{ id: string; name: string; index: number; map: MapType }>>([]);
  const [floorRoot, setFloorRoot] = useState<{ id: string; name: string } | null>(null);
  const [selectedFloorId, setSelectedFloorId] = useState<string | null>(null);

  const [planPins, setPlanPins] = useState<ApiPin[]>([]);
  const [areaPins, setAreaPins] = useState<ApiAreaPin[]>([]);

  const [mode, setMode] = useState<"edit" | "user">("edit");

  const [sideMenuMode, setSideMenuMode] = useState<"map" | "plan" | "area" | null>(null);
  const [editAreaTarget, setEditAreaTarget] = useState<{ id: string; initialName: string } | null>(null);
  const [editPlanTarget, setEditPlanTarget] = useState<ApiPin | null>(null);

  const openMapEdit = () => {
    if (!mapEditTarget) {
      alert("編集するマップを先に選択してください。");
      return;
    }
    setSideMenuMode("map");
  };
  const closeSideMenu = () => {
    setSideMenuMode(null);
    setPlacingKind(null);
    setDraftPos(null);
    setEditAreaTarget(null);
    setEditPlanTarget(null);
  };

  const [showPinKindModal, setShowPinKindModal] = useState(false);
  const [selectedKind, setSelectedKind] = useState<PinKind | null>(null);

  const [placingKind, setPlacingKind] = useState<PinKind | null>(null);
  const [draftPos, setDraftPos] = useState<{ xNorm: number; yNorm: number } | null>(null);

  const [isSheetOpen, setIsSheetOpen] = useState(false);
  const [selectedSpot, setSelectedSpot] = useState<SpotData | null>(null);

  const normalizeMap = useCallback((raw: any): MapType => {
    if (!raw) {
      return {
        id: "",
        name: "",
        imageData: null,
        naturalWidth: 0,
        naturalHeight: 0,
        parentMapId: null,
        hasFloors: false,
        floorCount: 0,
      };
    }
    const floorCount =
      typeof raw?.floorCount === "number" && !Number.isNaN(raw.floorCount)
        ? raw.floorCount
        : 0;

    return {
      id: raw.id ?? "",
      name: raw.name ?? "",
      imageData: raw.imageData ?? null,
      naturalWidth: raw.naturalWidth ?? 0,
      naturalHeight: raw.naturalHeight ?? 0,
      parentMapId: raw.parentMapId ?? null,
      hasFloors: floorCount > 0,
      floorCount,
    };
  }, []);

  const applyMapUpdate = useCallback((mapObj: MapType) => {
    if (!mapObj.id) return;
    setMaps((prev) => {
      const index = prev.findIndex((m) => m.id === mapObj.id);
      if (index === -1) {
        return [...prev, mapObj];
      }
      const merged = mergeMap(prev[index], mapObj);
      if (merged === prev[index]) {
        return prev;
      }
      const next = [...prev];
      next[index] = merged;
      return next;
    });
    setSelectedMap((prev) => {
      if (!prev || prev.id !== mapObj.id) return prev;
      return mergeMap(prev, mapObj);
    });
  }, []);

  const loadFloorStack = useCallback(
    async (mapId: string, preferredFloorId?: string | null) => {
      try {
        const res = await fetch(`${API}/maps/${mapId}/floors`, { credentials: "include" });
        if (!res.ok) {
          setFloorRoot(null);
          setFloorItems([]);
          setSelectedFloorId(null);
          return [];
        }
        const payload = await res.json();
        const rootId: string = typeof payload?.rootMapId === "string" && payload.rootMapId.length > 0
          ? payload.rootMapId
          : mapId;
        const rootNameRaw: string = typeof payload?.rootName === "string" ? payload.rootName : "";
        const rootName = rootNameRaw.trim();

        const rawItems: any[] = Array.isArray(payload?.items) ? payload.items : [];
        const floors = rawItems
          .map((item) => {
            const mapData = normalizeMap(item?.map);
            if (!mapData.id) return null;
            const index = Number.isFinite(item?.floorIndex) ? Number(item.floorIndex) : 0;
            return {
              id: mapData.id,
              name: mapData.name,
              index,
              map: mapData,
            };
          })
          .filter(
            (item): item is { id: string; name: string; index: number; map: MapType } =>
              !!item && item.id.length > 0 && item.index >= 0
          )
          .sort((a, b) => {
            if (a.index !== b.index) return a.index - b.index;
            return a.id.localeCompare(b.id);
          });

        const updatedCount =
          typeof payload?.floorCount === "number" ? payload.floorCount : floors.length;
        const hasFloorsFlag = updatedCount > 0;

        setFloorRoot(hasFloorsFlag ? { id: rootId, name: rootName } : null);
        setFloorItems(floors);

        const fallbackFloorId = floors[0]?.id ?? null;
        const explicitPreferred =
          preferredFloorId ?? (floors.some((f) => f.id === mapId) ? mapId : undefined);
        setSelectedFloorId((prev) => {
          const desiredId =
            (explicitPreferred && floors.some((f) => f.id === explicitPreferred))
              ? explicitPreferred
              : prev && floors.some((f) => f.id === prev)
                ? prev
                : fallbackFloorId;
          return desiredId ?? null;
        });

        setMaps((prev) => {
          let changed = false;
          let next = prev;
          const ensureCopy = () => {
            if (next === prev) {
              next = [...prev];
            }
          };
          const upsert = (patch: MapPatch) => {
            const idx = next.findIndex((m) => m.id === patch.id);
            if (idx === -1) {
              ensureCopy();
              next.push({
                id: patch.id,
                name: patch.name ?? "",
                imageData: patch.imageData ?? null,
                naturalWidth: patch.naturalWidth ?? 0,
                naturalHeight: patch.naturalHeight ?? 0,
                parentMapId: patch.parentMapId ?? null,
                hasFloors: patch.hasFloors ?? false,
                floorCount: patch.floorCount ?? 0,
              });
              changed = true;
            } else {
              const merged = mergeMap(next[idx], patch);
              if (merged !== next[idx]) {
                ensureCopy();
                next[idx] = merged;
                changed = true;
              }
            }
          };

          const rootPatch: MapPatch = {
            id: rootId,
            hasFloors: hasFloorsFlag,
            floorCount: updatedCount,
          };
          if (rootName) {
            rootPatch.name = rootName;
          }
          upsert(rootPatch);
          floors.forEach(({ map }) => {
            upsert(map);
          });

          return changed ? next : prev;
        });

        setSelectedMap((prev) => {
          if (!prev) return prev;
          if (prev.id === rootId) {
            const patch: MapPatch = {
              id: rootId,
              hasFloors: hasFloorsFlag,
              floorCount: updatedCount,
            };
          if (rootName) {
            patch.name = rootName;
          }
            return mergeMap(prev, patch);
          }
          const matchingFloor = floors.find((f) => f.id === prev.id);
          if (matchingFloor) {
            return mergeMap(prev, matchingFloor.map);
          }
          return prev;
        });

        return floors;
      } catch (err) {
        console.error("Failed to load floors:", err);
        setFloorRoot(null);
        setFloorItems([]);
        setSelectedFloorId(null);
        return [];
      }
    },
    [API, normalizeMap]
  );

  const activeFloor = useMemo(() => {
    if (!floorItems.length) return null;
    if (selectedFloorId) {
      const current = floorItems.find((f) => f.id === selectedFloorId);
      if (current) return current;
    }
    return floorItems[0];
  }, [floorItems, selectedFloorId]);

  const activeMap = useMemo(() => {
    if (activeFloor) return activeFloor.map;
    return selectedMap;
  }, [activeFloor, selectedMap]);

  const floorOptions = useMemo(
    () =>
      floorItems.map((f) => ({
        id: f.id,
        label:
          f.index > 0
            ? `${f.index}F`
            : f.name || f.map.name || f.id,
      })),
    [floorItems]
  );

  const topFloorItem = useMemo(() => {
    if (!floorItems.length) return null;
    return floorItems.reduce((prev, current) =>
      current.index > prev.index ? current : prev
    );
  }, [floorItems]);

  const floorRootId = useMemo(() => {
    if (floorRoot?.id) return floorRoot.id;
    if (!selectedMap) return null;
    if (selectedMap.hasFloors) return selectedMap.id;
    if (selectedMap.parentMapId) {
      const parent = maps.find((m) => m.id === selectedMap.parentMapId);
      if (parent?.hasFloors) return parent.id;
      return selectedMap.id;
    }
    return null;
  }, [floorRoot?.id, maps, selectedMap]);

  const stairAddDisabled =
    !floorRootId || (!selectedMap?.hasFloors && !selectedMap?.parentMapId);
  const stairDeleteDisabled =
    !floorRootId || !topFloorItem || selectedFloorId !== topFloorItem.id;
  const mapEditTarget = activeMap ?? selectedMap;

  const displayTitle = useMemo(() => {
    if (floorItems.length > 0 && activeFloor) {
      let baseName = floorRoot?.name ?? "";
      if (!baseName && floorRoot?.id) {
        const rootEntry = maps.find((m) => m.id === floorRoot.id);
        if (rootEntry?.name) {
          baseName = rootEntry.name;
        }
      }
      if (!baseName) {
        baseName = selectedMap?.name ?? "未選択";
      }
      const floorNumber = activeFloor.index;
      if (Number.isFinite(floorNumber) && floorNumber > 0) {
        return `${baseName}/${floorNumber}階`;
      }
      return baseName;
    }
    return activeMap?.name ?? selectedMap?.name ?? "未選択";
  }, [activeFloor, activeMap?.name, floorItems.length, floorRoot, maps, selectedMap?.name]);

  // 初期ロード：マップ一覧
  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${API}/maps/index`, { credentials: "include" });
        const data = await res.json();
        const rawItems: any[] =
          Array.isArray(data?.items) ? data.items :
          Array.isArray(data) ? data :
          data ? [data] : [];
        const normalized = rawItems
          .map((item) => normalizeMap(item))
          .filter((item) => item.id);
        setMaps(normalized);
        setSelectedMap((prev) => prev ?? (normalized.length > 0 ? normalized[0] : null));
      } catch (e) {
        console.error("Failed to fetch maps:", e);
      }
    })();
  }, [normalizeMap]);

  useEffect(() => {
    if (!selectedMap) {
      setFloorRoot(null);
      setFloorItems([]);
      setSelectedFloorId(null);
      return;
    }

    const parentMap = selectedMap.parentMapId
      ? maps.find((m) => m.id === selectedMap.parentMapId)
      : null;

    if (selectedMap.hasFloors) {
      setFloorItems([]);
      setSelectedFloorId(null);
      setFloorRoot(null);
      void loadFloorStack(selectedMap.id);
      return;
    }

    if (parentMap?.hasFloors) {
      setFloorItems([]);
      setSelectedFloorId(selectedMap.id);
      setFloorRoot(null);
      void loadFloorStack(parentMap.id, selectedMap.id);
      return;
    }

    setFloorRoot(null);
    setFloorItems([]);
    setSelectedFloorId(null);
  }, [loadFloorStack, maps, selectedMap]);

  // ピン一覧
  useEffect(() => {
    const targetMapId = activeMap?.id ?? null;
    if (!targetMapId) {
      setPlanPins([]);
      setAreaPins([]);
      return;
    }
    setPlanPins([]);
    setAreaPins([]);
    (async () => {
      try {
        const res = await fetch(`${API}/maps/${targetMapId}/pins`, { credentials: "include" });
        if (!res.ok) {
          console.warn("Failed to fetch pins:", res.status);
          setPlanPins([]); setAreaPins([]); return;
        }
        const pins = (await res.json()) as any[];
        const plans: ApiPin[] = [];
        const areas: ApiAreaPin[] = [];
        for (const p of pins) {
          if (p.type === "area_selector") {
            areas.push({
              id: p.id, mapId: p.mapId, name: p.name,
              xNorm: p.xNorm, yNorm: p.yNorm, linkToMapId: p.linkToMapId ?? null,
            });
          } else {
            plans.push({
              id: p.id, mapId: p.mapId, name: p.name,
              description: p.description ?? null,
              descriptionImageData: p.descriptionImageData ?? null,
              type: p.type, linkToMapId: p.linkToMapId ?? null,
              xNorm: p.xNorm, yNorm: p.yNorm,
              place: p.place ?? null,
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
        setPlanPins([]); setAreaPins([]);
      }
    })();
  }, [activeMap?.id]);

  useEffect(() => {
    setDraftPos(null);
  }, [activeMap?.id]);

  // 「＋ピン」
  const handleOpenPinKindModal = () => { setSelectedKind(null); setShowPinKindModal(true); };
  const handleConfirmPinKind = () => {
    if (!selectedKind) return;
    setPlacingKind(selectedKind);
    setDraftPos(null);
    setShowPinKindModal(false);
  };

  // MapImage 内クリックで座標確定
  const handleAddPinAt = (xNorm: number, yNorm: number) => {
    if (draftPos) return;
    if (!placingKind || !activeMap) return;
    setDraftPos({ xNorm, yNorm });
    setSideMenuMode(placingKind); // "area" or "plan"
  };

  const handleSelectFloor = useCallback((floorId: string) => {
    setSelectedFloorId(floorId);
  }, []);

  const handleStairAdd = useCallback(async () => {
    const targetId = floorRoot?.id ?? selectedMap?.id ?? null;
    if (!targetId) {
      alert("階を追加するマップが選択されていません。");
      return;
    }
    try {
      const res = await fetch(`${API}/maps/${encodeURIComponent(targetId)}/floors`, {
        method: "POST",
        credentials: "include",
      });
      if (!res.ok) {
        const text = await res.text();
        alert(`階の追加に失敗しました: ${res.status} ${text}`);
        return;
      }
      const created = await res.json();
      const newFloorId: string | null = created?.id ?? null;
      const highestIndex = floorItems.reduce((max, f) => Math.max(max, f.index), 0);
      const nextIndex = highestIndex + 1;
      const nextName = `${nextIndex}F`;
      if (newFloorId) {
        try {
          await fetch(`${API}/maps/${encodeURIComponent(newFloorId)}`, {
            method: "PATCH",
            headers: { "Content-Type": "application/json" },
            credentials: "include",
            body: JSON.stringify({ name: nextName }),
          });
        } catch (err) {
          console.warn("階名の更新に失敗しました:", err);
        }
      }
      await loadFloorStack(targetId, newFloorId);
    } catch (err) {
      console.error(err);
      alert("階の追加に失敗しました。ログを確認してください。");
    }
  }, [API, floorItems, floorRoot?.id, loadFloorStack, selectedMap?.id]);

  const handleStairDelete = useCallback(async () => {
    const targetId = floorRoot?.id ?? selectedMap?.id ?? null;
    if (!targetId) {
      alert("階を削除するマップが選択されていません。");
      return;
    }
    if (!topFloorItem) {
      alert("削除できる階がありません。");
      return;
    }
    if (selectedFloorId !== topFloorItem.id) {
      alert("削除できるのは一番上の階のみです。");
      return;
    }
    if (!confirm(`「${topFloorItem.index}階」を削除しますか？`)) return;
    try {
      const res = await fetch(
        `${API}/maps/${encodeURIComponent(targetId)}/floors/${topFloorItem.index}`,
        {
          method: "DELETE",
          credentials: "include",
        }
      );
      if (!res.ok) {
        const text = await res.text();
        alert(`階の削除に失敗しました: ${res.status} ${text}`);
        return;
      }
      await loadFloorStack(targetId);
    } catch (err) {
      console.error(err);
      alert("階の削除に失敗しました。ログを確認してください。");
    }
  }, [API, floorRoot?.id, loadFloorStack, selectedFloorId, selectedMap?.id, topFloorItem]);

  // PlanPin 選択
  const handlePlanPinSelect = (spot: SpotData & { id?: string }) => {
    if (mode === "edit") {
      // 念のため、ボトムシートを閉じてからサイドメニューに切替
      setIsSheetOpen(false);
      setSelectedSpot(null);

      const t = spot.id ? planPins.find((p) => p.id === spot.id) : undefined;
      if (t) {
        setEditPlanTarget(t);
        setSideMenuMode("plan");
        return;
      }
    }
    // Userモード：従来どおりボトムシート
    setSelectedSpot(spot);
    setIsSheetOpen(true);
  };

  // AreaPin 選択（モード別）
  const handleAreaPinSelect = async (area: ApiAreaPin) => {
    if (mode === "edit") {
      setEditAreaTarget({ id: area.id, initialName: area.name });
      setSideMenuMode("area");
      return;
    }
    if (!area.linkToMapId) return;
    await goToMapId(area.linkToMapId);
  };

  // 親マップへ戻る
  const handleBackToParent = async () => {
    const parentId = selectedMap?.parentMapId;
    if (!parentId) return;
    await goToMapId(parentId);
  };

  // マップ遷移（必要時 GET /maps/{id}）
  const goToMapId = async (mapId: string) => {
    const existing = maps.find((m) => m.id === mapId);
    if (existing) { setSelectedMap(existing); return; }
    try {
      const res = await fetch(`${API}/maps/${mapId}`, { credentials: "include" });
      if (!res.ok) {
        const fallback = {
          id: mapId,
          name: "",
          imageData: null,
          naturalWidth: 0,
          naturalHeight: 0,
          parentMapId: null,
          hasFloors: false,
          floorCount: 0,
        };
        applyMapUpdate(fallback);
        setSelectedMap(fallback);
        return;
      }
      const m = await res.json();
      const mapObj = normalizeMap(m);
      applyMapUpdate(mapObj);
      setSelectedMap(mapObj);
    } catch {
      const fallback = {
        id: mapId,
        name: "",
        imageData: null,
        naturalWidth: 0,
        naturalHeight: 0,
        parentMapId: null,
        hasFloors: false,
        floorCount: 0,
      };
      applyMapUpdate(fallback);
      setSelectedMap(fallback);
    }
  };

  // 作成/更新/削除のローカル反映
  const appendAreaPin = (p: ApiAreaPin) => setAreaPins((prev) => [...prev, p]);
  const updateAreaPinLocal = (p: ApiAreaPin) =>
    setAreaPins((prev) => prev.map((x) => (x.id === p.id ? { ...x, name: p.name } : x)));
  const removeAreaPinLocal = (pinId: string) =>
    setAreaPins((prev) => prev.filter((x) => x.id !== pinId));

  const appendPlanPin = (p: ApiPin) => setPlanPins((prev) => [...prev, p]);
  const updatePlanPinLocal = (p: ApiPin) =>
    setPlanPins((prev) => prev.map((x) => (x.id === p.id ? { ...x, ...p } : x)));
  const removePlanPinLocal = (pinId: string) =>
    setPlanPins((prev) => prev.filter((x) => x.id !== pinId));

  const headerNode = useMemo(
    () => (
      <AdminHeader
        currentMapName={displayTitle}
        mode={mode}
        onModeChange={(m) => setMode(m)}
        onMapEdit={openMapEdit}
        onStairAdd={handleStairAdd}
        onStairDelete={handleStairDelete}
        disableStairAdd={stairAddDisabled}
        disableStairDelete={stairDeleteDisabled}
      />
    ),
    [
      displayTitle,
      handleStairAdd,
      handleStairDelete,
      mode,
      openMapEdit,
      stairAddDisabled,
      stairDeleteDisabled,
    ]
  );

  return (
    <div className="flex flex-col h-screen bg-[#f5f0dc]">
      <MapHeader
        mapName={selectedMap?.name ?? ""}
        parentMapId={selectedMap?.parentMapId ?? null}
        onBack={handleBackToParent}
      />

      <div className="relative flex-1 min-h-0 flex">
        <Map
          pins={planPins}
          areaPins={areaPins}
          mode={mode}
          onPlanPinSelect={handlePlanPinSelect}
          onAreaPinSelect={handleAreaPinSelect}
          onAddPin={mode === "edit" && activeMap ? handleOpenPinKindModal : undefined}
          onAddPinAt={placingKind && activeMap ? handleAddPinAt : undefined}
          placing={!!placingKind}
          placingKind={placingKind}
          draftPos={draftPos}
          mapId={activeMap?.id ?? null}
          mapImageData={toPreviewUrl(activeMap?.imageData ?? null)}
          naturalWidth={activeMap?.naturalWidth ?? 4096}
          naturalHeight={activeMap?.naturalHeight ?? 3072}
          floors={floorOptions}
          selectedFloorId={selectedFloorId}
          onSelectFloor={handleSelectFloor}
          header={headerNode}
        />
      </div>

      {sideMenuMode && (
        <SideMenu
          mode={sideMenuMode}
          onClose={closeSideMenu}
          mapEditProps={
            sideMenuMode === "map" && mapEditTarget
              ? {
                  mapId: mapEditTarget.id,
                  initialName: mapEditTarget.name,
                  initialImageUrl: toPreviewUrl(mapEditTarget.imageData ?? null),
                  onSaved: async (updated) => {
                    const normalized = normalizeMap(updated);
                    applyMapUpdate(normalized);
                    setFloorItems((prev) =>
                      prev.map((f) =>
                        f.id === normalized.id
                          ? {
                              ...f,
                              name: normalized.name || f.name,
                              map: { ...f.map, ...normalized },
                            }
                          : f
                      )
                    );
                    setFloorRoot((prev) =>
                      prev && prev.id === normalized.id
                        ? { ...prev, name: normalized.name ?? prev.name }
                        : prev
                    );
                    try {
                      const res = await fetch(`${API}/maps/${normalized.id}`, { credentials: "include" });
                      if (res.ok) {
                        const fresh = normalizeMap(await res.json());
                        applyMapUpdate(fresh);
                        setFloorItems((prev) =>
                          prev.map((f) =>
                            f.id === fresh.id
                              ? {
                                  ...f,
                                  name: fresh.name || f.name,
                                  map: { ...f.map, ...fresh },
                                }
                              : f
                          )
                        );
                        setFloorRoot((prev) =>
                          prev && prev.id === fresh.id
                            ? { ...prev, name: fresh.name ?? prev.name }
                            : prev
                        );
                      }
                    } catch (err) {
                      console.warn("Failed to refresh map info after save:", err);
                    }
                    if (normalized.parentMapId || selectedMap?.id === mapEditTarget.id) {
                      const parentId = normalized.parentMapId ?? selectedMap?.id ?? null;
                      if (parentId) {
                        await loadFloorStack(parentId, normalized.id);
                      }
                    }
                    closeSideMenu();
                  },
                  onDeleted: async (parentMapId) => {
                    if (parentMapId) {
                      await goToMapId(parentMapId);
                      await loadFloorStack(parentMapId);
                    } else {
                      setFloorItems([]);
                      setSelectedFloorId(null);
                    }
                    closeSideMenu();
                  },
                  onClose: closeSideMenu,
                }
              : undefined
          }
          pinContext={{
            placingKind: placingKind,
            mapId: activeMap?.id ?? null,
            draftPos: draftPos,
            onAreaCreated: appendAreaPin,
            onPlanCreated: appendPlanPin,
          }}
          editAreaPin={editAreaTarget}
          onAreaUpdated={updateAreaPinLocal}
          onAreaDeleted={removeAreaPinLocal}
          // プラン編集の受け渡し
          editPlanPin={editPlanTarget}
          onPlanUpdated={(p) => {
            updatePlanPinLocal(p);
          }}
          onPlanDeleted={(pinId) => {
            removePlanPinLocal(pinId);
          }}
        />
      )}

      <PinKindSelectModal
        visible={showPinKindModal}
        selected={selectedKind}
        onSelect={(kind) => setSelectedKind(kind as PinKind)}
        onConfirm={handleConfirmPinKind}
        onCancel={() => { setShowPinKindModal(false); setSelectedKind(null); }}
      />

      <PlanSpotBottomSheet isOpen={isSheetOpen} spotData={selectedSpot} onClose={() => setIsSheetOpen(false)} />
    </div>
  );
}
