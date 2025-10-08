"use client";
import React from "react";
import styles from "./Button.module.css";

type Props = {
  onClick?: () => void;
  disabled?: boolean;
};

export const DeleteButton: React.FC<Props> = ({ onClick, disabled }) => (
  <button
    type="button"
    className={styles.deleteButton}
    onClick={onClick}
    disabled={disabled}
    aria-disabled={disabled}
  >
    削除
  </button>
);
