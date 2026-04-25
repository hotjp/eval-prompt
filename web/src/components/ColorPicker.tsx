
const PRESET_COLORS = [
  { value: 'default', label: '—', hex: '#00000020' },
  { value: 'red', label: '', hex: '#ff4d4f' },
  { value: 'orange', label: '', hex: '#ff7a45' },
  { value: 'gold', label: '', hex: '#ffc53d' },
  { value: 'lime', label: '', hex: '#a0d911' },
  { value: 'green', label: '', hex: '#52c41a' },
  { value: 'cyan', label: '', hex: '#13c2c2' },
  { value: 'blue', label: '', hex: '#1890ff' },
  { value: 'purple', label: '', hex: '#722ed1' },
  { value: 'geekblue', label: '', hex: '#2f54eb' },
  { value: 'magenta', label: '', hex: '#eb2f96' },
  { value: 'volcano', label: '', hex: '#fa541c' },
]

interface ColorPickerProps {
  color?: string
  value?: string
  onChange?: (color: string) => void
}

function ColorPicker({ color, value, onChange }: ColorPickerProps) {
  const selected = color || value || 'default'

  return (
    <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
      {PRESET_COLORS.map(({ value: c, label, hex }) => (
        <div
          key={c}
          onClick={() => onChange?.(c)}
          style={{
            width: 32,
            height: 32,
            borderRadius: 4,
            background: hex,
            border: selected === c ? '2px solid #1890ff' : '1px solid #d9d9d9',
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            fontSize: 11,
            color: c === 'default' ? '#666' : '#fff',
            fontWeight: selected === c ? 700 : 400,
            boxShadow: selected === c ? '0 0 0 2px rgba(24,144,255,0.2)' : 'none',
            transition: 'all 0.15s',
          }}
          title={c}
        >
          {label}
        </div>
      ))}
    </div>
  )
}

export default ColorPicker
