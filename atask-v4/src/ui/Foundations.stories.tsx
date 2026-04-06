import type { Meta, StoryObj } from "@storybook/react-vite";
import Surface from "./Surface";

const meta = {
  title: "Design System/Foundations",
  tags: ["autodocs"],
} satisfies Meta;

export default meta;
type Story = StoryObj<typeof meta>;

const colors = [
  ["Canvas", "var(--canvas)"],
  ["Canvas Elevated", "var(--canvas-elevated)"],
  ["Accent", "var(--accent)"],
  ["Accent Subtle", "var(--accent-subtle)"],
  ["Ink Primary", "var(--ink-primary)"],
  ["Ink Secondary", "var(--ink-secondary)"],
  ["Deadline", "var(--deadline-red)"],
  ["Success", "var(--success)"],
];

export const Tokens: Story = {
  render: () => (
    <Surface style={{ width: 720 }}>
      <div className="ui-story-grid">
        <div>
          <div className="view-label">Color Roles</div>
          <div className="ui-story-swatch-grid">
            {colors.map(([label, value]) => (
              <div key={label} className="ui-story-swatch">
                <div className="ui-story-color" style={{ background: value }} />
                <div className="ui-story-meta">
                  <div className="ui-story-label">{label}</div>
                  <div className="ui-story-token">{value}</div>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div>
          <div className="view-label">Spacing Scale</div>
          <div className="ui-story-grid">
            {["--sp-2", "--sp-3", "--sp-4", "--sp-6", "--sp-8"].map((token) => (
              <div key={token} style={{ display: "grid", gap: "var(--sp-2)" }}>
                <div className="ui-story-token">{token}</div>
                <div style={{ width: `var(${token})`, height: 10, background: "var(--accent)" }} />
              </div>
            ))}
          </div>
        </div>
      </div>
    </Surface>
  ),
};
