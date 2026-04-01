import type { Meta, StoryObj } from "@storybook/react-vite";
import MenuList from "./MenuList";
import Surface from "./Surface";

const meta = {
  title: "Design System/MenuList",
  component: MenuList,
  tags: ["autodocs"],
} satisfies Meta<typeof MenuList>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    items: [
      { label: "Rename", shortcut: "R" },
      { label: "Archive" },
      { separator: true },
      { label: "Delete", danger: true },
    ],
  },
  render: (args) => (
    <Surface padded={false} style={{ width: 260 }}>
      <MenuList {...args} activeIndex={0} />
    </Surface>
  ),
};
