import type { Meta, StoryObj } from '@storybook/react-vite';
import PopoverPanel from './PopoverPanel';

const meta = {
  title: 'UI/PopoverPanel',
  component: PopoverPanel,
  args: {
    title: 'Move to Project',
    children: (
      <>
        <button type="button" className="ui-picker-row">
          <span className="ui-picker-label">Inbox</span>
        </button>
        <div className="ui-popover-separator" />
        <div className="ui-picker-input-wrap">
          <input className="ui-picker-input" placeholder="Search..." />
        </div>
      </>
    ),
  },
} satisfies Meta<typeof PopoverPanel>;

export default meta;
type Story = StoryObj<typeof meta>;

export const Default: Story = {};
