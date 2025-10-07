export default function AddPinButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}
      className="absolute bottom-6 right-6 bg-yellow-600 text-white w-12 h-12 rounded-full shadow-lg text-2xl"
    >
      ＋
    </button>
  );
}
