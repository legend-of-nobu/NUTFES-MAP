"use client";
import React, { useLayoutEffect, useRef, useState } from "react";

export type ApiAreaPin = {
  id: string;
  mapId: string;
  name: string;
  xNorm: number; // 0..1
  yNorm: number; // 0..1
  linkToMapId?: string | null;
};

type Props = {
  area: ApiAreaPin;
  onSelect: (area: ApiAreaPin) => void;
  ghost?: boolean; // ★ 追加
};

const MAX_FONT_SIZE = 12; // px
const MIN_FONT_SIZE = 5;  // px

const AreaPin: React.FC<Props> = ({ area, onSelect, ghost = false }) => {
  const [fontSize, setFontSize] = useState(MAX_FONT_SIZE);
  const containerRef = useRef<HTMLDivElement>(null);
  const textRef = useRef<HTMLSpanElement>(null);

  const formattedName = area.name.split("\n").map((line, index, arr) => (
    <React.Fragment key={index}>
      {line}
      {index < arr.length - 1 && <br />}
    </React.Fragment>
  ));

  useLayoutEffect(() => {
    const container = containerRef.current;
    const text = textRef.current;
    if (!container || !text) return;

    const tempText = text.cloneNode(true) as HTMLSpanElement;
    tempText.style.position = "absolute";
    tempText.style.visibility = "hidden";
    tempText.style.wordBreak = "break-all";
    document.body.appendChild(tempText);

    let current = MAX_FONT_SIZE;
    tempText.style.fontSize = `${current}px`;

    if (
      tempText.scrollHeight > container.clientHeight ||
      tempText.scrollWidth > container.clientWidth
    ) {
      while (current > MIN_FONT_SIZE) {
        current--;
        tempText.style.fontSize = `${current}px`;
        if (
          tempText.scrollHeight <= container.clientHeight &&
          tempText.scrollWidth <= container.clientWidth
        ) {
          break;
        }
      }
    }

    document.body.removeChild(tempText);
    setFontSize(current);
  }, [area.name]);

  const posStyle = {
    left: `${area.xNorm * 100}%`,
    top: `${area.yNorm * 100}%`,
  };

  const interactiveProps = ghost
    ? {}
    : {
        onClick: (e: React.MouseEvent) => {
          e.stopPropagation();
          onSelect(area);
        },
      };

  return (
    <div
      style={{ position: "absolute", ...posStyle }}
      className={`absolute -translate-x-1/2 -translate-y-full ${
        ghost ? "pointer-events-none opacity-60" : "cursor-pointer"
      }`}
      role={ghost ? undefined : "button"}
      tabIndex={ghost ? -1 : 0}
      aria-label={ghost ? undefined : `${area.name}（エリア）`}
      {...interactiveProps}
    >
      <div className="relative h-[67px] w-[80px]">
        <img src="/fan.svg" alt="エリア" className="absolute left-0 top-0 h-full w-full" />
        <div
          ref={containerRef}
          className="absolute left-1/2 top-[40%] flex h-[40%] w-[70%] -translate-x-1/2 -translate-y-1/2 items-center justify-center"
        >
          <span
            ref={textRef}
            className="pointer-events-none text-center font-bold text-white"
            style={{ fontSize: `${fontSize}px`, lineHeight: "1.2", wordBreak: "break-all" }}
          >
            {formattedName}
          </span>
        </div>
      </div>
    </div>
  );
};

export default AreaPin;
