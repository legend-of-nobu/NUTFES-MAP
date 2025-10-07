// src/components/map/InteractiveMap.tsx
'use client';

import React from 'react';
import { useRouter } from 'next/navigation';

type Props = { mapId: string | null };

export default function InteractiveMap({ mapId }: Props) {
  const router = useRouter();
  // 既存の表示・スタイルはそのまま．内部処理のみ必要に応じて実装
  return <div className="w-full h-full" />;
}
