"use client";
import React from "react";
import  "../../Image.css"; 
import "../../FormStyle.css";
import "../../Style.css"


type PreviewPlanImageProps = {
  image: string;
};

export default function PreviewPlanImage({ image }: PreviewPlanImageProps) {
  return (
    <div className="previewContainer">
      <img src={image} alt="企画画像プレビュー" className="preview" />
    </div>
  );
}
