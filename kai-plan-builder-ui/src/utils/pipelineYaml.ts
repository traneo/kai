import type { BuilderStep, PipelineConfig, StepPolicy } from '../types'

export function defaultPolicy(): StepPolicy {
  return {
    allowedDirs: [],
    agent: '',
    allowedTools: [],
    allowedCommands: [],
    maxRetries: 0,
    timeoutSeconds: 0,
    retryDelaySeconds: 0,
    retryBackoff: 'linear',
    saveState: false,
  }
}

export function newStep(id: string): BuilderStep {
  return {
    id,
    prompt: '',
    dependsOn: [],
    validation: [],
    approval: 'optional',
    policy: defaultPolicy(),
  }
}

export function parseYaml(yaml: string): PipelineConfig {
  const lines = yaml.split('\n')
  const config: PipelineConfig = {
    project: '',
    repoURL: '',
    repoBaseBranch: 'main',
    repoProvider: '',
    repoTokenRef: '',
    outputType: 'pr',
    branchPrefix: 'feat/',
    steps: [],
  }

  let i = 0
  let currentStep: Partial<BuilderStep> | null = null
  let inPrompt = false
  let promptLines: string[] = []
  let promptIndent = 0

  function getIndent(line: string): number {
    return line.search(/\S|$/)
  }

  function finishPrompt() {
    if (currentStep && inPrompt) {
      currentStep.prompt = promptLines.join('\n')
      promptLines = []
      inPrompt = false
    }
  }

  function finishStep() {
    if (currentStep && currentStep.id) {
      const policy = currentStep.policy || defaultPolicy()
      config.steps.push({
        id: currentStep.id || '',
        prompt: currentStep.prompt || '',
        dependsOn: currentStep.dependsOn || [],
        validation: currentStep.validation || [],
        approval: currentStep.approval || 'optional',
        policy: {
          allowedDirs: policy.allowedDirs || [],
          agent: policy.agent || '',
          allowedTools: policy.allowedTools || [],
          allowedCommands: policy.allowedCommands || [],
          maxRetries: policy.maxRetries || 0,
          timeoutSeconds: policy.timeoutSeconds || 0,
          retryDelaySeconds: policy.retryDelaySeconds || 0,
          retryBackoff: policy.retryBackoff || 'linear',
          saveState: policy.saveState || false,
        },
      })
    }
    currentStep = null
    inPrompt = false
    promptLines = []
  }

  for (; i < lines.length; i++) {
    const raw = lines[i]
    const line = raw.replace(/#.*$/, '').trimEnd()

    if (!line.trim() || line.trim().startsWith('#')) {
      if (inPrompt && raw.includes(promptLines[promptLines.length - 1] || '')) {
        promptLines.push('')
      }
      continue
    }

    if (inPrompt) {
      const indent = getIndent(raw)
      if (indent >= promptIndent) {
        promptLines.push(raw.slice(promptIndent))
        continue
      } else {
        finishPrompt()
      }
    }

    const indent = getIndent(raw)
    const trimmed = line.trim()

    if (trimmed === 'version: 1') continue
    if (trimmed === 'version: "1"') continue
    if (trimmed === "version: '1'") continue

    const colonIdx = trimmed.indexOf(':')
    if (colonIdx === -1) continue

    let key = trimmed.slice(0, colonIdx).trim()
    let val = trimmed.slice(colonIdx + 1).trim()

    if (key.startsWith('- ')) {
      key = key.slice(2).trim()
    }

    if (key === 'project' && indent === 0) {
      config.project = val
    } else if (key === 'url' && indent === 2) {
      config.repoURL = val
    } else if (key === 'base_branch' && indent === 2) {
      config.repoBaseBranch = val
    } else if (key === 'provider' && indent === 2) {
      config.repoProvider = val
    } else if (key === 'token_ref' && indent === 2) {
      config.repoTokenRef = val
    } else if (key === 'type' && indent === 2) {
      config.outputType = val
    } else if (key === 'branch_prefix' && indent === 2) {
      config.branchPrefix = val
    } else if (key === 'id' && indent === 2) {
      finishStep()
      currentStep = { id: val, policy: defaultPolicy() }
    } else if (key === 'prompt' && indent === 4) {
      if (val === '|') {
        inPrompt = true
        promptLines = []
        promptIndent = indent + 2
      } else if (val) {
        if (currentStep) currentStep.prompt = val
      }
    } else if (key === 'depends_on') {
      if (val.startsWith('[') && val.endsWith(']')) {
        if (currentStep) currentStep.dependsOn = val.slice(1, -1).split(',').map(s => s.trim()).filter(Boolean)
      } else {
        const deps: string[] = []
        i++
        while (i < lines.length) {
          const depRaw = lines[i].trim()
          if (!depRaw || depRaw.startsWith('#')) { i++; continue }
          const depIndent = getIndent(lines[i])
          if (depIndent <= 4) break
          if (depRaw.startsWith('- ')) {
            deps.push(depRaw.slice(2).trim())
          }
          i++
        }
        i--
        if (currentStep) currentStep.dependsOn = deps
      }
    } else if (key === 'validation') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const vRaw = lines[i].trim()
        if (!vRaw || vRaw.startsWith('#')) { i++; continue }
        const vIndent = getIndent(lines[i])
        if (vIndent <= 4) break
        if (vRaw.startsWith('- ')) {
          vals.push(vRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) currentStep.validation = vals
    } else if (key === 'approval' && indent === 3) {
      if (currentStep) currentStep.approval = val
    } else if (key === 'max_retries') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.maxRetries = parseInt(val) || 0
      }
    } else if (key === 'timeout_seconds') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.timeoutSeconds = parseInt(val) || 0
      }
    } else if (key === 'retry_delay_seconds') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.retryDelaySeconds = parseInt(val) || 0
      }
    } else if (key === 'retry_backoff') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.retryBackoff = val
      }
    } else if (key === 'save_state') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.saveState = val === 'true'
      }
    } else if (key === 'allowed_dirs') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const dRaw = lines[i].trim()
        if (!dRaw || dRaw.startsWith('#')) { i++; continue }
        const dIndent = getIndent(lines[i])
        if (dIndent <= 5) break
        if (dRaw.startsWith('- ')) {
          vals.push(dRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.allowedDirs = vals
      }
    } else if (key === 'agent') {
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.agent = lines[i].substring(key.length + 1).trim()
      }
    } else if (key === 'allowed_tools') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const tRaw = lines[i].trim()
        if (!tRaw || tRaw.startsWith('#')) { i++; continue }
        const tIndent = getIndent(lines[i])
        if (tIndent <= 5) break
        if (tRaw.startsWith('- ')) {
          vals.push(tRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.allowedTools = vals
      }
    } else if (key === 'allowed_commands') {
      const vals: string[] = []
      i++
      while (i < lines.length) {
        const cRaw = lines[i].trim()
        if (!cRaw || cRaw.startsWith('#')) { i++; continue }
        const cIndent = getIndent(lines[i])
        if (cIndent <= 5) break
        if (cRaw.startsWith('- ')) {
          vals.push(cRaw.slice(2).trim())
        }
        i++
      }
      i--
      if (currentStep) {
        if (!currentStep.policy) currentStep.policy = defaultPolicy()
        currentStep.policy.allowedCommands = vals
      }
    }
  }

  finishPrompt()
  finishStep()

  return config
}

export function serializeYaml(config: PipelineConfig): string {
  const indent = (n: number) => '  '.repeat(n)

  let yaml = `version: 1\n`
  yaml += `project: ${config.project}\n`

  if (config.repoURL) {
    yaml += `repo:\n`
    yaml += `${indent(1)}url: ${config.repoURL}\n`
    if (config.repoBaseBranch) {
      yaml += `${indent(1)}base_branch: ${config.repoBaseBranch}\n`
    }
    if (config.repoProvider) {
      yaml += `${indent(1)}provider: ${config.repoProvider}\n`
    }
    if (config.repoTokenRef) {
      yaml += `${indent(1)}token_ref: ${config.repoTokenRef}\n`
    }
  }

  yaml += `output:\n`
  yaml += `${indent(1)}type: ${config.outputType}\n`
  if (config.branchPrefix) {
    yaml += `${indent(1)}branch_prefix: ${config.branchPrefix}\n`
  }

  yaml += `steps:\n`
  for (const step of config.steps) {
    yaml += `${indent(1)}- id: ${step.id}\n`
    yaml += `${indent(2)}prompt: |\n`
    for (const line of step.prompt.split('\n')) {
      yaml += `${indent(3)}${line || ''}\n`
    }
    if (step.dependsOn.length > 0) {
      yaml += `${indent(2)}depends_on:\n`
      for (const dep of step.dependsOn) {
        yaml += `${indent(3)}- ${dep}\n`
      }
    }
    if (step.validation.length > 0) {
      yaml += `${indent(2)}validation:\n`
      for (const v of step.validation) {
        yaml += `${indent(3)}- ${v}\n`
      }
    }
    if (step.approval) {
      yaml += `${indent(2)}approval: ${step.approval}\n`
    }

    const p = step.policy
    const hasPolicy = p.maxRetries > 0 || p.timeoutSeconds > 0 || p.retryDelaySeconds > 0
      || p.allowedDirs.length > 0 || p.agent !== '' || p.allowedTools.length > 0
      || p.allowedCommands.length > 0 || p.retryBackoff !== 'linear' || p.saveState

    if (hasPolicy) {
      yaml += `${indent(2)}policy:\n`
      for (const dir of p.allowedDirs) yaml += `${indent(3)}allowed_dirs:\n${indent(4)}- ${dir}\n`
      if (p.agent) yaml += `${indent(3)}agent: ${p.agent}\n`
      for (const tool of p.allowedTools) yaml += `${indent(3)}allowed_tools:\n${indent(4)}- ${tool}\n`
      for (const cmd of p.allowedCommands) yaml += `${indent(3)}allowed_commands:\n${indent(4)}- ${cmd}\n`
      if (p.maxRetries > 0) yaml += `${indent(3)}max_retries: ${p.maxRetries}\n`
      if (p.timeoutSeconds > 0) yaml += `${indent(3)}timeout_seconds: ${p.timeoutSeconds}\n`
      if (p.retryDelaySeconds > 0) yaml += `${indent(3)}retry_delay_seconds: ${p.retryDelaySeconds}\n`
      if (p.retryBackoff !== 'linear') yaml += `${indent(3)}retry_backoff: ${p.retryBackoff}\n`
      if (p.saveState) yaml += `${indent(3)}save_state: true\n`
    }
  }

  return yaml
}

export function layoutDAG(steps: BuilderStep[]): BuilderStep[][] {
  const depths = new Map<string, number>()

  function getDepth(id: string): number {
    if (depths.has(id)) return depths.get(id)!
    const s = steps.find(x => x.id === id)
    if (!s || s.dependsOn.length === 0) {
      depths.set(id, 0)
      return 0
    }
    let maxDep = 0
    for (const dep of s.dependsOn) {
      maxDep = Math.max(maxDep, getDepth(dep) + 1)
    }
    depths.set(id, maxDep)
    return maxDep
  }

  for (const s of steps) getDepth(s.id)
  const maxDepth = Math.max(...Array.from(depths.values()), 0)
  const levels: BuilderStep[][] = Array.from({ length: maxDepth + 1 }, () => [])
  for (const s of steps) {
    levels[depths.get(s.id)!].push(s)
  }
  return levels
}
