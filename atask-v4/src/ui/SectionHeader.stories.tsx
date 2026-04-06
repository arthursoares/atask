import type { Meta, StoryObj } from '@storybook/react-vite';
import SectionHeader from './SectionHeader';

const meta = {
  title: 'UI/SectionHeader',
  component: SectionHeader,
  args: {
    title: 'This Evening',
    muted: true,
  },
} satisfies Meta<typeof SectionHeader>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const Collapsible: Story = {
  args: {
    title: 'Backlog',
    count: 12,
    collapsible: true,
    collapsed: false,
  },
};
