interface DateGroupHeaderProps {
  date: string;
  relative?: string;
}

export default function DateGroupHeader({ date, relative }: DateGroupHeaderProps) {
  return (
    <div className="date-group-header">
      <span>{date}</span>
      {relative && <span className="relative">{relative}</span>}
    </div>
  );
}
