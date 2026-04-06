import type { HTMLAttributes, ReactNode } from "react";

type SurfaceTone = "default" | "sunken" | "accent";

interface SurfaceProps extends HTMLAttributes<HTMLDivElement> {
  children: ReactNode;
  tone?: SurfaceTone;
  padded?: boolean;
}

export default function Surface({
  children,
  tone = "default",
  padded = true,
  className,
  ...props
}: SurfaceProps) {
  const classes = [
    "ui-surface",
    `ui-surface-${tone}`,
    padded ? "ui-surface-padded" : "",
    className ?? "",
  ].filter(Boolean).join(" ");

  return (
    <div className={classes} {...props}>
      {children}
    </div>
  );
}
