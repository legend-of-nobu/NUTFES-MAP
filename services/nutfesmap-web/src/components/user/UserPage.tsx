"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import MapHeader from "../admin/MapHeader";
import Map from "../admin/Map";
import PlanSpotBottomSheet from "@/components/map/PlanSpotBottomSheet";
import { toPreviewUrl } from "../admin/SideMenu/MapEditForm/base64";
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

const API = process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

export default function UserPage() {
  const [maps, setMaps] = useState<MapType[]>([]);
  const [selectedMap, setSelectedMap] = useState<MapType | null>(null);
  const [floorRoot, setFloorRoot] = useState<{ id: string; name: string } | null>(null);
  const [floorItems, setFloorItems] = useState<
    Array<{ id: string; name: string; index: number; map: MapType }>
  >([]);
  const [selectedFloorId, setSelectedFloorId] = useState<string | null>(null);

  const [planPins, setPlanPins] = useState<ApiPin[]>([]);
  const [areaPins, setAreaPins] = useState<ApiAreaPin[]>([]);

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
        label: f.index > 0 ? `${f.index}F` : f.name || f.map.name || f.id,
      })),
    [floorItems]
  );

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

  const goToMapId = useCallback(
    async (mapId: string) => {
      const existing = maps.find((m) => m.id === mapId);
      if (existing) {
        setSelectedMap(existing);
        return;
      }
      try {
        const res = await fetch(`${API}/maps/${mapId}`, { credentials: "include" });
        if (!res.ok) {
          const fallback: MapType = {
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
        const fallback: MapType = {
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
    },
    [applyMapUpdate, maps, normalizeMap]
  );

  const handlePlanPinSelect = (spot: SpotData & { id?: string }) => {
    setSelectedSpot(spot);
    setIsSheetOpen(true);
  };

  const handleAreaPinSelect = async (area: ApiAreaPin) => {
    if (!area.linkToMapId) return;
    await goToMapId(area.linkToMapId);
  };

  const handleBackToParent = async () => {
    const parentId = selectedMap?.parentMapId;
    if (!parentId) return;
    await goToMapId(parentId);
  };

  const handleSelectFloor = useCallback((floorId: string) => {
    setSelectedFloorId(floorId);
  }, []);

  return (
    <div className="flex flex-col h-screen bg-[#f5f0dc]">
      <MapHeader
        mapName={displayTitle}
        parentMapId={selectedMap?.parentMapId ?? null}
        onBack={handleBackToParent}
      />

      <div className="relative flex-1 min-h-0 flex">
        <Map
          pins={planPins}
          areaPins={areaPins}
          mode="user"
          onPlanPinSelect={handlePlanPinSelect}
          onAreaPinSelect={handleAreaPinSelect}
          mapId={activeMap?.id ?? null}
          mapImageData={toPreviewUrl(activeMap?.imageData ?? null)}
          naturalWidth={activeMap?.naturalWidth ?? 4096}
          naturalHeight={activeMap?.naturalHeight ?? 3072}
          floors={floorOptions}
          selectedFloorId={selectedFloorId}
          onSelectFloor={handleSelectFloor}
        />
      </div>

      <PlanSpotBottomSheet
        isOpen={isSheetOpen}
        spotData={selectedSpot}
        onClose={() => setIsSheetOpen(false)}
      />
    </div>
  );
}
