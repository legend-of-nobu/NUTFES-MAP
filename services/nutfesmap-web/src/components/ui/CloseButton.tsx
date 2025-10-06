export default function CloseButton({ onClick }: { onClick: () => void }) {
  return (
    <button onClick={onClick} className="px-3 py-1 border rounded hover:bg-gray-100">
      閉じる
    </button>
  );
}
