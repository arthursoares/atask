import type { Meta, StoryObj } from "@storybook/react-vite";
import Field from "./Field";
import Surface from "./Surface";

const meta = {
  title: "Design System/Field",
  component: Field,
  tags: ["autodocs"],
} satisfies Meta<typeof Field>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {
  args: {
    label: "Project title",
    placeholder: "Ship v1",
    hint: "Used in project lists and command palette results.",
  },
};

export const States: Story = {
  render: () => (
    <Surface style={{ width: 320 }}>
      <div className="ui-story-grid">
        <Field label="Default" defaultValue="Weekly review" />
        <Field label="Placeholder" placeholder="Inbox capture" />
        <Field label="Error" defaultValue="Someday maybe" error="Name must be at least 3 characters." />
      </div>
    </Surface>
  ),
};
