import { FaCirclePlus} from "react-icons/fa6";
import"tailwindcss";
export default function AddPinButton({ onClick }: { onClick: () => void }) {
  return (
    <button
      onClick={onClick}>
      <FaCirclePlus className="absolute bottom-6 right-6 bg-white text-main w-14 h-14 rounded-full shadow-lg text-2xl"/>
    </button>
  );
}
