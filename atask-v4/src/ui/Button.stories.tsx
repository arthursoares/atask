import type { Meta, StoryObj } from "@storybook/react-vite";
import Button from "./Button";
import Surface from "./Surface";

const meta = {
  title: "Design System/Button",
  component: Button,
  args: {
    children: "Save changes",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof Button>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Primary: Story = {
  args: {
    variant: "primary",
  },
};

export const AllVariants: Story = {
  render: () => (
    <Surface>
      <div style={{ display: "flex", gap: "var(--sp-3)", flexWrap: "wrap" }}>
        <Button variant="primary">Primary</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="danger">Danger</Button>
      </div>
    </Surface>
  ),
};

export const Sizes: Story = {
  render: () => (
    <Surface>
      <div style={{ display: "flex", alignItems: "center", gap: "var(--sp-3)", flexWrap: "wrap" }}>
        <Button size="sm">Small</Button>
        <Button size="default">Default</Button>
        <Button size="lg">Large</Button>
      </div>
    </Surface>
  ),
};
