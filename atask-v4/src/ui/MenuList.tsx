import type { ReactNode } from "react";

export type MenuListItem =
  | {
      label: string;
      icon?: ReactNode;
      shortcut?: string;
      danger?: boolean;
      disabled?: boolean;
      separator?: false;
      onClick?: () => void;
    }
  | { separator: true };

interface MenuListProps {
  items: MenuListItem[];
  activeIndex?: number;
  onItemHover?: (index: number) => void;
  onItemLeave?: () => void;
  onItemClick?: (item: MenuListItem) => void;
}

export default function MenuList({
  items,
  activeIndex = -1,
  onItemHover,
  onItemLeave,
  onItemClick,
}: MenuListProps) {
  return (
    <div className="ui-menu-list">
      {items.map((item, i) => {
        if ("separator" in item) {
          return <div key={i} className="ui-menu-separator" />;
        }

        const isActive = activeIndex === i;

        return (
          <button
            key={i}
            type="button"
            className={[
              "ui-menu-item",
              isActive ? "is-active" : "",
              item.danger ? "is-danger" : "",
            ].filter(Boolean).join(" ")}
            disabled={item.disabled}
            onMouseEnter={() => onItemHover?.(i)}
            onMouseLeave={() => onItemLeave?.()}
            onClick={() => onItemClick?.(item)}
          >
            {item.icon && <span className="ui-menu-icon">{item.icon}</span>}
            <span className="ui-menu-label">{item.label}</span>
            {item.shortcut && <span className="ui-menu-shortcut">{item.shortcut}</span>}
          </button>
        );
      })}
    </div>
  );
}
