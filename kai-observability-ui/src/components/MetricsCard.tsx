interface Props {
  label: string
  value: string | number
  sub?: string
  color?: string
}

export function MetricsCard({ label, value, sub, color }: Props) {
  return (
    <div className="metrics-card" style={color ? { borderTopColor: color } : undefined}>
      <div className="metrics-card-label">{label}</div>
      <div className="metrics-card-value">{value}</div>
      {sub && <div className="metrics-card-sub">{sub}</div>}
    </div>
  )
}
