"use client";
import React from "react";
import { HiMiniXCircle } from "react-icons/hi2";
import styles from "./Button.module.css";

export const CloseButton: React.FC<{ onClick: () => void }> = ({ onClick }) => (
  <button className={styles.closeButton} onClick={onClick}>
    <HiMiniXCircle size={40} />
  </button>
);




