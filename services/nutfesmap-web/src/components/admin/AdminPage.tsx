'use client';
import { useEffect, useState } from 'react';
import AdminHeader from './AdminHeader';
import MapHeader from './MapHeader';
import Map from './Map';
import PinKindSelectModal from './PinKindSelectModal';

type MapType = { id: string; name: string; imageData?: string | null };
type PinType = { id: string; name: string; xNorm: number; yNorm: number };

export default function AdminPage() {
  const [maps, setMaps] = useState<MapType[]>([]);
  const [selectedMap, setSelectedMap] = useState<MapType | null>(null);
  const [pins, setPins] = useState<PinType[]>([]);
  const [showModal, setShowModal] = useState(false);
  const [mode, setMode] = useState<'edit' | 'user'>('edit');
  const [selectedKind, setSelectedKind] = useState<'area' | 'plan' | null>(null);

  useEffect(() => {
    fetch(`${process.env.NEXT_PUBLIC_API_BASE_URL}/maps/index`)
      .then(res => res.json())
      .then(data => setMaps(data.items))
      .catch(err => console.error(err));
  }, []);

  return (
    <div className="flex flex-col h-screen bg-[#f5f0dc]">
      <MapHeader mapName={selectedMap?.name ?? ''} onBack={() => setSelectedMap(null)} />
      <AdminHeader
        currentMapName={selectedMap?.name ?? '未選択'}
        mode={mode}
        onModeChange={() => setMode(mode === 'edit' ? 'user' : 'edit')}
        onMapEdit={() => console.log('マップ編集')}
        onStairEdit={() => console.log('階層編集')}
      />

      <Map pins={pins} onPinClick={() => {}} onAddPin={() => setShowModal(true)} />

      <PinKindSelectModal
        visible={showModal}
        selected={selectedKind}
        onSelect={(kind) => setSelectedKind(kind)} // ← 選択だけ
        onConfirm={() => {
          console.log('選択確定:', selectedKind);
          setShowModal(false); // ← 決定ボタンで閉じる
        }}
        onCancel={() => setShowModal(false)} // ← 閉じるボタンで閉じる
      />
    </div>
  );
}
