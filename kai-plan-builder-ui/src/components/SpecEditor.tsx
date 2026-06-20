interface Props {
  value: string
  onChange: (value: string) => void
}

export function SpecEditor({ value, onChange }: Props) {
  return (
    <textarea
      className="spec-editor"
      value={value}
      onChange={e => onChange(e.target.value)}
      placeholder={`Start by describing your pipeline project...

You can type directly here or use the chat panel on the right to build your spec through conversation.

Example:
# Project: my-service
# Repo: github.com/org/my-service
# Steps:
#   1. Lint and typecheck
#   2. Run tests
#   3. Build and deploy`}
      spellCheck={false}
    />
  )
}
