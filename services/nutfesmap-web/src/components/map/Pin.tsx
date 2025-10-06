export default function Pin({ pin, onClick }: any) {
  return (
    <div
      onClick={onClick}
      style={{ left: `${pin.xNorm * 100}%`, top: `${pin.yNorm * 100}%` }}
      className="absolute transform -translate-x-1/2 -translate-y-1/2 cursor-pointer"
    >
      <div className="bg-yellow-700 w-4 h-4 rounded-full border-2 border-white shadow" />
    </div>
  );
}
