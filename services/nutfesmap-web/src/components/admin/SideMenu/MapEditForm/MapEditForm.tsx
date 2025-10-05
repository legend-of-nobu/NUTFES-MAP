import React, { useState } from "react";
import { MapNameForm } from "./MapNameForm";
import { MapImageUploadForm } from "./MapImageUploadForm";
import PreviewMapImage from "./PreviewMapImage";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import "../Style.css";
import "../Image.css";

export const MapEditForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const [mapName, setMapName] = useState("");
  const [mapImage, setMapImage] = useState<File | null>(null);

  const image = mapImage ? URL.createObjectURL(mapImage) : null;

  return (
    <div className="container">
      <CloseButton onClick={onClose} />
      <h2 className="title"> マップを編集</h2>
      <MapNameForm value={mapName} onChange={setMapName} />
      <label className="Label">マップ画像</label>
      <div className="fileinputContainer">
        <MapImageUploadForm value={mapImage} onChange={setMapImage} />
        {image && (
        <div>
          <PreviewMapImage image={image} /> 
        </div> )}
      </div>
      <div className="buttonRow">
      <SaveButton />
      <DeleteButton />
      </div>
    </div>
  );
};
