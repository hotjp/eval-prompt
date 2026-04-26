// Shared category configuration
// Category determines the asset's type: Normal (content), Eval Case, or Metric

export const categoryLabels: Record<string, string> = {
  content: 'Normal',
  eval: 'Eval Case',
  metric: 'Metric',
}

export const categoryOptions = [
  { label: 'Normal', value: 'content' },
  { label: 'Eval Case', value: 'eval' },
  { label: 'Metric', value: 'metric' },
]

export const categoryColors: Record<string, string> = {
  content: 'blue',
  eval: 'purple',
  metric: 'cyan',
}
