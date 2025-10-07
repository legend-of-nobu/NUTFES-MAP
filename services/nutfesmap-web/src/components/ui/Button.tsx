type Props = {
  label: string;
  onClick: () => void;
  color?: "yellow" | "green" | "red";
  icon?: React.ReactNode;
};

export function Button({
  label,
  onClick,
  color = "yellow",
  icon,
}: Props) {
  const base =
    "w-[120px] py-1.5 rounded-full border text-sm font-medium transition flex items-center justify-center gap-1";
  const colors = {
    yellow:
      "border-yellow-700 text-yellow-700 hover:bg-yellow-50 bg-white",
    green:
      "border-green-500 text-green-500 hover:bg-green-50 bg-white",
    red: "border-red-500 text-red-500 hover:bg-red-50 bg-white",
  };

  return (
    <button onClick={onClick} className={`${base} ${colors[color]}`}>
      {icon && <span className="text-sm">{icon}</span>}
      {label}
    </button>
  );
}
