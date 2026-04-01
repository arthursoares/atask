import type { Meta, StoryObj } from '@storybook/react-vite';
import TagPill from './TagPill';

const meta = {
  title: 'UI/TagPill',
  component: TagPill,
  args: {
    label: 'Design',
    variant: 'default',
  },
} satisfies Meta<typeof TagPill>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const Today: Story = { args: { label: 'Today', variant: 'today' } };
export const Deadline: Story = { args: { label: 'Overdue', variant: 'deadline' } };
export const Removable: Story = { args: { onRemove: () => {} } };
