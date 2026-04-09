import { useEffect, useRef } from 'react';
import { useStore } from '@nanostores/react';
import { $locations, setTaskLocation } from '../store';
import { PopoverPanel } from '../ui';

interface LocationPickerProps {
  taskId: string;
  currentLocationId: string | null;
  onClose: () => void;
}

/**
 * Lightweight location picker. Lists every known location with a check
 * mark next to the currently-selected one, plus a "No location" entry to
 * clear the association. Creating new locations still happens from the
 * Locations view — the picker's job is just assignment.
 */
export default function LocationPicker({ taskId, currentLocationId, onClose }: LocationPickerProps) {
  const locations = useStore($locations);
  const popoverRef = useRef<HTMLDivElement>(null);

  // Click-outside to close
  useEffect(() => {
    const handleMouseDown = (e: MouseEvent) => {
      if (popoverRef.current && !popoverRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    document.addEventListener('mousedown', handleMouseDown);
    return () => document.removeEventListener('mousedown', handleMouseDown);
  }, [onClose]);

  // Escape closes the picker without propagating to any surrounding
  // panel-level Esc handler.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onClose();
      }
    };
    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [onClose]);

  const handleSelect = async (locationId: string | null) => {
    await setTaskLocation(taskId, locationId);
    onClose();
  };

  return (
    <PopoverPanel title="Location" className="location-popover" popoverRef={popoverRef}>
      <div
        className={`when-option${currentLocationId === null ? ' selected' : ''}`}
        onClick={() => handleSelect(null)}
      >
        <span className="when-icon">—</span>
        <span>No location</span>
        {currentLocationId === null && <span className="when-check">✓</span>}
      </div>
      {locations.length === 0 ? (
        <div className="location-empty-hint">
          No locations yet. Create one from the Locations view.
        </div>
      ) : (
        locations.map((loc) => {
          const isSelected = currentLocationId === loc.id;
          return (
            <div
              key={loc.id}
              className={`when-option${isSelected ? ' selected' : ''}`}
              onClick={() => handleSelect(loc.id)}
            >
              <span className="when-icon">📍</span>
              <span>{loc.name}</span>
              {isSelected && <span className="when-check">✓</span>}
            </div>
          );
        })
      )}
    </PopoverPanel>
  );
}
