import type { Meta, StoryObj } from '@storybook/react-vite';
import ProgressBar from './ProgressBar';

const meta = {
  title: 'UI/ProgressBar',
  component: ProgressBar,
  args: {
    completed: 3,
    total: 7,
  },
} satisfies Meta<typeof ProgressBar>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
export const Complete: Story = { args: { completed: 7, total: 7 } };
