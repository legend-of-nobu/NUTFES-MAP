import React, { useState, useRef, useLayoutEffect } from "react";

type PlanPinProps = {
  name: string;
  onClick: () => void;
};

// --- 設定値 ---
const MAX_FONT_SIZE = 12; // テキストの最大フォントサイズ(px)
const MIN_FONT_SIZE = 5; // テキストの最小フォントサイズ(px)
// --- ---

const PlanPin: React.FC<PlanPinProps> = ({ name, onClick }) => {
  const [fontSize, setFontSize] = useState(MAX_FONT_SIZE);
  const containerRef = useRef<HTMLDivElement>(null);
  const textRef = useRef<HTMLSpanElement>(null);

  const formattedName = name.split("\n").map((line, index, arr) => (
    <React.Fragment key={index}>
      {line}
      {index < arr.length - 1 && <br />}
    </React.Fragment>
  ));

  useLayoutEffect(() => {
    const container = containerRef.current;
    const text = textRef.current;
    if (!container || !text) return;

    // --- 計算ロジックの修正 ---
    // 画面外でサイズ計算するためのクローン要素を作成
    const tempText = text.cloneNode(true) as HTMLSpanElement;
    tempText.style.position = "absolute";
    tempText.style.visibility = "hidden";
    tempText.style.wordBreak = "break-all";
    document.body.appendChild(tempText);

    let currentSize = MAX_FONT_SIZE;
    tempText.style.fontSize = `${currentSize}px`;

    // 最大フォントサイズではみ出すかチェックし、はみ出す場合のみループで最適なサイズを探す
    if (
      tempText.scrollHeight > container.clientHeight ||
      tempText.scrollWidth > container.clientWidth
    ) {
      while (currentSize > MIN_FONT_SIZE) {
        currentSize--;
        tempText.style.fontSize = `${currentSize}px`;
        if (
          tempText.scrollHeight <= container.clientHeight &&
          tempText.scrollWidth <= container.clientWidth
        ) {
          break; // 収まるサイズが見つかったらループを抜ける
        }
      }
    }

    // 計算が終わったらクローン要素を削除し、算出したフォントサイズを適用する
    document.body.removeChild(tempText);
    setFontSize(currentSize);
    // --------------------------
  }, [name]); // nameプロパティが変更されたときに再計算する

  return (
    <div
      onClick={onClick}
      className="relative cursor-pointer w-[80px] h-[67px] -translate-x-1/2 -translate-y-full"
    >
      <img
        src="/fan.svg"
        alt="企画ピン"
        className="absolute top-0 left-0 w-full h-full"
      />
      <div
        ref={containerRef}
        className="absolute top-[40%] left-1/2 -translate-x-1/2 -translate-y-1/2 w-[70%] h-[40%] flex items-center justify-center"
      >
        <span
          ref={textRef}
          className="text-white font-bold text-center pointer-events-none"
          style={{
            fontSize: `${fontSize}px`,
            lineHeight: "1.2",
            wordBreak: "break-all",
          }}
        >
          {formattedName}
        </span>
      </div>
    </div>
  );
};

export default PlanPin;
