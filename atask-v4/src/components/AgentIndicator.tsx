interface AgentIndicatorProps {
  status: string;
}

export default function AgentIndicator({ status }: AgentIndicatorProps) {
  return (
    <span className="task-agent">✦ {status}</span>
  );
}
