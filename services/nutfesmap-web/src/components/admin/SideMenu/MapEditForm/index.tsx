import React, { useState } from "react";
import { MapNameForm } from "./MapNameForm";
import { MapImageUploadForm } from "./MapImageUploadForm";
import PreviewMapImage from "./PreviewMapImage";
import { SaveButton } from "../CommonButton/SaveButton";
import { DeleteButton } from "../CommonButton/DeleteButton";
import { CloseButton } from "../CommonButton/CloseButton";
import styles from "./MapEditForm.module.css";

export const MapEditForm: React.FC<{ onClose: () => void }> = ({ onClose }) => {
  const [mapName, setMapName] = useState("");
  const [mapImage, setMapImage] = useState<File | null>(null);

  const image = mapImage ? URL.createObjectURL(mapImage) : null;

  return (
    <div className={styles.container}>
      <CloseButton onClick={onClose} />
      <h2 className={styles.title}> マップを編集</h2>
      <MapNameForm value={mapName} onChange={setMapName} />
      <label className={styles.mapImageLabel}>マップ画像</label>
      <div className={styles.fileinput}>
        <MapImageUploadForm value={mapImage} onChange={setMapImage} />
        {image && (
        <div className={styles.previewWrapper}>
          <PreviewMapImage image={image} /> 
        </div> )}
      </div>
      <div className={styles.buttonRow}>
      <SaveButton />
      <DeleteButton />
      </div>
    </div>
  );
};
