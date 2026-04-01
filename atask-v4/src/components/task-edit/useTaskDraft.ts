import { useEffect, useRef, useState } from 'react';
import { updateTask } from '../../store';

interface UseTaskDraftOptions {
  taskId: string;
  title: string;
  notes: string;
  debounceMs?: number;
}

export default function useTaskDraft({
  taskId,
  title,
  notes,
  debounceMs = 300,
}: UseTaskDraftOptions) {
  const [titleValue, setTitleValue] = useState(title);
  const [notesValue, setNotesValue] = useState(notes);
  const titleTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const notesTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const prevTaskIdRef = useRef(taskId);

  useEffect(() => {
    if (prevTaskIdRef.current !== taskId) {
      prevTaskIdRef.current = taskId;
      setTitleValue(title);
      setNotesValue(notes);
    }
  }, [taskId, title, notes]);

  useEffect(() => {
    return () => {
      if (titleTimerRef.current) clearTimeout(titleTimerRef.current);
      if (notesTimerRef.current) clearTimeout(notesTimerRef.current);
    };
  }, []);

  const handleTitleChange = (value: string) => {
    setTitleValue(value);
    if (titleTimerRef.current) clearTimeout(titleTimerRef.current);
    titleTimerRef.current = setTimeout(() => {
      updateTask({ id: taskId, title: value });
    }, debounceMs);
  };

  const handleNotesChange = (value: string) => {
    setNotesValue(value);
    if (notesTimerRef.current) clearTimeout(notesTimerRef.current);
    notesTimerRef.current = setTimeout(() => {
      updateTask({ id: taskId, notes: value });
    }, debounceMs);
  };

  const flushDraft = () => {
    if (titleTimerRef.current) clearTimeout(titleTimerRef.current);
    if (notesTimerRef.current) clearTimeout(notesTimerRef.current);
    updateTask({ id: taskId, title: titleValue.trim(), notes: notesValue });
  };

  const clearPendingDraft = () => {
    if (titleTimerRef.current) clearTimeout(titleTimerRef.current);
    if (notesTimerRef.current) clearTimeout(notesTimerRef.current);
  };

  return {
    titleValue,
    notesValue,
    setTitleValue: handleTitleChange,
    setNotesValue: handleNotesChange,
    flushDraft,
    clearPendingDraft,
  };
}
