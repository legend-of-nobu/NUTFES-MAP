"use client";
import React from "react";
import "../Image.css"; 
import "../Style.css"

type PreviewMapImageProps = {
  image: string;
};

export default function PreviewMapImage({ image }: PreviewMapImageProps) {
  return (
    <div className="previewContainer">
      <img src={image} alt="マップ画像プレビュー" className="preview" />
    </div>
  );
}
