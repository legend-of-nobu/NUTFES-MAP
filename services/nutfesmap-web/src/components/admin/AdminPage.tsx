'use client';
import { useEffect, useState } from 'react';
import AdminHeader from './AdminHeader';
import MapHeader from './MapHeader';
import Map from './Map';
import PinKindSelectModal from './PinKindSelectModal';
// ⬇ 追加：SideMenu のインポート（あなたの構成に合わせてどちらか）
import SideMenu from './SideMenu/SideMenu';
// import SideMenu from './SideMenu'; // もし直下にある場合はこちら

type MapType = { id: string; name: string; imageData?: string | null };
type PinType = { id: string; name: string; xNorm: number; yNorm: number };

export default function AdminPage() {
  const [maps, setMaps] = useState<MapType[]>([]);
  const [selectedMap, setSelectedMap] = useState<MapType | null>(null);
  const [pins, setPins] = useState<PinType[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [mode, setMode] = useState<'edit' | 'user'>('edit');
  const [selectedKind, setSelectedKind] = useState<'area' | 'plan' | null>(null);

  // ⬇ 追加：SideMenu のモード（null で閉じる）
  const [sideMenuMode, setSideMenuMode] = useState<'plan' | 'area' | 'map' | null>(null);

  useEffect(() => {
    fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/maps/index`)
      .then(res => res.json())
      .then(data => setMaps(data.items))
      .catch(err => console.error(err));
  }, []);

  return (
    <div className="flex flex-col h-screen bg-[#f5f0dc]">
      <MapHeader mapName={selectedMap?.name ?? ''} onBack={() => setSelectedMap(null)} />

      <Map
        pins={pins}
        mode={mode}
        onPinClick={() => {}}
        onAddPin={() => setShowModal(true)}
        header={
          <AdminHeader
            currentMapName={selectedMap?.name ?? '未選択'}
            mode={mode}
            // ✅ 修正：引数 m を受けてそのままセット（ModeChanger の呼び出し仕様に一致）
            onModeChange={(m) => setMode(m)}
            // ✅ 追加：“マップ編集”で SideMenu を map モードで開く
            onMapEdit={() => setSideMenuMode('map')}
            onStairAdd={() => console.log('階を追加')}
            onStairDelete={() => console.log('階を削除')}
          />
        }
      />

      {/* ⬇ 追加：SideMenu の実体（内部 CSS/見た目は SideMenu 側に任せる） */}
      {mode === 'edit' && sideMenuMode && (
        <SideMenu mode={sideMenuMode} onClose={() => setSideMenuMode(null)} />
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
