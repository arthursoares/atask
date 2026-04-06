import type { Meta, StoryObj } from '@storybook/react-vite';
import EmptyState from './EmptyState';

const icon = (
  <svg viewBox="0 0 48 48" style={{ width: 48, height: 48 }}>
    <circle cx="24" cy="24" r="16" fill="none" stroke="currentColor" strokeWidth="2" />
    <path d="M17 24h14" stroke="currentColor" strokeWidth="2" />
  </svg>
);

const meta = {
  title: 'UI/EmptyState',
  component: EmptyState,
  args: {
    icon,
    text: 'Nothing here yet',
  },
} satisfies Meta<typeof EmptyState>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
