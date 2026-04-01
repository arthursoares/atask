import { useState } from 'react';

export default function useTaskPickers() {
  const [showWhenPicker, setShowWhenPicker] = useState(false);
  const [showTagPicker, setShowTagPicker] = useState(false);
  const [showRepeatPicker, setShowRepeatPicker] = useState(false);
  const [showProjectPicker, setShowProjectPicker] = useState(false);

  return {
    showWhenPicker,
    setShowWhenPicker,
    showTagPicker,
    setShowTagPicker,
    showRepeatPicker,
    setShowRepeatPicker,
    showProjectPicker,
    setShowProjectPicker,
  };
}
