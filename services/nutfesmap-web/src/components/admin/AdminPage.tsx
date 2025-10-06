'use client';
import { useEffect, useState } from 'react';
import AdminHeader from './AdminHeader';
import MapHeader from './MapHeader';
import Map from './Map';
import PinKindSelectModal from './PinKindSelectModal';
import SideMenu from './SideMenu/SideMenu';

type MapType = { id: string; name: string; imageData?: string | null };
type PinType = { id: string; name: string; xNorm: number; yNorm: number };

export default function AdminPage() {
  const [maps, setMaps] = useState<MapType[]>([]);
  const [selectedMap, setSelectedMap] = useState<MapType | null>(null);
  const [pins, setPins] = useState<PinType[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [mode, setMode] = useState<'edit' | 'user'>('edit');
  const [selectedKind, setSelectedKind] = useState<'area' | 'plan' | null>(null);

  // SideMenu のモード（null で閉じる）
  const [sideMenuMode, setSideMenuMode] = useState<'plan' | 'area' | 'map' | null>(null);

  useEffect(() => {
    (async () => {
      try {
        const res = await fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/maps/index`, {
          credentials: 'include',
        });
        const data = await res.json();
        const items: MapType[] = Array.isArray(data?.items) ? data.items : [];
        setMaps(items);

        // 初期選択：まだ何も選ばれていなければ先頭を選ぶ（上書き防止）
        setSelectedMap((prev) => prev ?? (items.length > 0 ? items[0] : null));
      } catch (err) {
        console.error(err);
      }
    })();
  }, []);

  // “マップ編集”を押したとき、選択中マップがなければ SideMenu を開かない
  const openMapEdit = () => {
    if (!selectedMap) {
      alert('編集するマップを先に選択してください。');
      return;
    }
    setSideMenuMode('map');
  };

  // 保存後の反映：maps & selectedMap を更新し、SideMenu を閉じる
  const handleMapSaved = (updated: MapType) => {
    setMaps((prev) => prev.map((m) => (m.id === updated.id ? { ...m, ...updated } : m)));
    setSelectedMap((prev) => (prev && prev.id === updated.id ? { ...prev, ...updated } : prev));
    setSideMenuMode(null);
  };

  return (
    <div className="flex flex-col h-screen bg-[#f5f0dc]">
      <MapHeader mapName={selectedMap?.name ?? ''} onBack={() => setSelectedMap(null)} />

      <Map
        pins={pins}
        mode={mode}
        onPinClick={() => {}}
        onAddPin={() => setShowModal(true)}
        // Map 背景を即時反映したい場合は、Map 側に mapImageData を使わせる
        // mapId を内部で利用して再取得する実装でもOK
        // @ts-ignore 既存の Map プロップに合わせてください
        mapId={selectedMap?.id ?? null}
        // @ts-ignore 既存の Map プロップに合わせてください
        mapImageData={selectedMap?.imageData ?? null}
        header={
          <AdminHeader
            currentMapName={selectedMap?.name ?? '未選択'}
            mode={mode}
            onModeChange={(m) => setMode(m)}
            onMapEdit={openMapEdit}
            onStairAdd={() => console.log('階を追加')}
            onStairDelete={() => console.log('階を削除')}
          />
        }
      />

      {/* SideMenu の実体（見た目は SideMenu 側に任せる） */}
      {mode === 'edit' && sideMenuMode && (
        <SideMenu
          mode={sideMenuMode}
          onClose={() => setSideMenuMode(null)}
          // MapEditForm へ選択中のマップ情報を受け渡し
          mapEditProps={
            sideMenuMode === 'map' && selectedMap
              ? {
                  mapId: selectedMap.id,
                  initialName: selectedMap.name,
                  initialImageUrl: selectedMap.imageData ?? null,
                  onSaved: handleMapSaved,
                }
              : undefined
          }
        />
      )}

      <PinKindSelectModal
        visible={showModal}
        selected={selectedKind}
        onSelect={(kind) => setSelectedKind(kind)}
        onConfirm={() => {
          console.log('選択確定:', selectedKind);
          setShowModal(false);
        }}
        onCancel={() => setShowModal(false)}
      />
    </div>
  );
}
