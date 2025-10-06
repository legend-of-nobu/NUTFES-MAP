import "tailwindcss";

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
    "w-[120px] py-1.5 rounded-[12px] border-[3px] text-sm font-family transition flex items-center justify-center gap-1";
  const colors = {
    yellow:
      "border-[#A08702] text-main hover:bg-yellow-50 bg-[#FFFCFC]",
    green:
      "border-[#75D070] text-[#75D070] hover:bg-green-50 bg-[#FFFCFC]",
    red: "border-[#B02F30] text-accent hover:bg-red-50 bg-[#FFFCFC]",
  };

  return (
    <button onClick={onClick} className={`${base} ${colors[color]}`}>
      {icon && <span className="text-sm">{icon}</span>}
      {label}
    </button>
  );
}
