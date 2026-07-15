interface GoEnum {
  name: string
  values: string[]
}

const GO_SOURCE_URL = new URL('../../internal/domain/enums.go', import.meta.url)
const GLOSSARY_URL = new URL('../src/shared/i18n/glossary.ts', import.meta.url)
const OUTPUT_URL = new URL('../src/shared/api/generated/enums.ts', import.meta.url)

const GO_ENUM_TO_GLOSSARY_DIMENSION: Readonly<Record<string, string>> = {
  ProfileType: 'profileType',
  IdentityType: 'identityType',
  DemandKind: 'demandKind',
  CaptureMode: 'captureMode',
  RecipientInputState: 'recipientInputState',
  RoutingDisposition: 'routingDisposition',
  FulfillmentLineReason: 'lineReason',
  SubmissionMode: 'submissionMode',
  SupplierOrderStatus: 'supplierOrderStatus',
  ShipmentStatus: 'shipmentStatus',
  AdjustmentKind: 'adjustmentKind',
  AllocationState: 'allocationState',
  AddressState: 'addressState',
  SupplierState: 'supplierState',
  ChannelSyncState: 'channelSyncState',
  LifecycleStage: 'lifecycleStage',
  ProductKind: 'productKind',
}

const NON_DOMAIN_GLOSSARY_DIMENSIONS = new Set([
  'driftSummary',
  'reviewRequirement',
  'basisDriftStatus',
  'allocationSelectorType',
  'demandMappingBlockedReason',
  'initialAllocationStrategy',
  'identityStrategy',
  'entitlementAuthorityMode',
  'recipientInputMode',
  'referenceStrategy',
  'trackingSyncMode',
  'closurePolicy',
  'documentType',
  'channelSyncJobStatus',
  'channelSyncItemStatus',
  'closureDecisionKind',
])

function fail(message: string): never {
  console.error(`gen-enums: ${message}`)
  Deno.exit(1)
}

function parseGoEnums(source: string): GoEnum[] {
  const lines = source.replaceAll('\r\n', '\n').split('\n')
  const enums: GoEnum[] = []

  for (let index = 0; index < lines.length; index++) {
    const declaration = /^type ([A-Za-z_][A-Za-z0-9_]*) string\s*$/.exec(lines[index])
    if (!declaration) continue

    const name = declaration[1]
    let cursor = index + 1
    while (cursor < lines.length && (lines[cursor].trim() === '' || lines[cursor].trimStart().startsWith('//'))) {
      cursor++
    }
    if (lines[cursor]?.trim() !== 'const (') {
      fail(`${name} at enums.go:${index + 1} is not followed by a const block`)
    }

    const values: string[] = []
    cursor++
    for (; cursor < lines.length && lines[cursor].trim() !== ')'; cursor++) {
      const line = lines[cursor]
      if (line.trim() === '' || line.trimStart().startsWith('//')) continue

      const entry = /^\s*([A-Za-z_][A-Za-z0-9_]*)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*("(?:\\.|[^"\\])*")\s*(?:\/\/.*)?$/.exec(line)
      if (!entry) {
        fail(`cannot confidently parse ${name} const entry at enums.go:${cursor + 1}: ${line.trim()}`)
      }
      if (entry[2] !== name) {
        fail(`expected const type ${name} at enums.go:${cursor + 1}, found ${entry[2]}`)
      }

      let value: string
      try {
        value = JSON.parse(entry[3])
      } catch {
        fail(`invalid string literal for ${name} at enums.go:${cursor + 1}`)
      }
      values.push(value)
    }

    if (cursor >= lines.length) fail(`unterminated const block for ${name}`)
    if (values.length === 0) fail(`typed const block for ${name} has no values`)
    if (new Set(values).size !== values.length) fail(`typed const block for ${name} contains duplicate wire values`)

    enums.push({ name, values })
    index = cursor
  }

  if (enums.length === 0) fail('no string-typed Go enums found')
  return enums
}

function arrayName(typeName: string): string {
  return typeName[0].toLowerCase() + typeName.slice(1) + 'Values'
}

function tsStringLiteral(value: string): string {
  return `'${value.replaceAll('\\', '\\\\').replaceAll("'", "\\'")}'`
}

function generate(enums: readonly GoEnum[]): string {
  const sections = enums.map((goEnum) => {
    const valuesName = arrayName(goEnum.name)
    const values = goEnum.values.map((value) => `  ${tsStringLiteral(value)},`).join('\n')
    return `export const ${valuesName} = [\n${values}\n] as const\n\nexport type ${goEnum.name} = (typeof ${valuesName})[number]`
  })

  return [
    '// DO NOT EDIT. Generated from internal/domain/enums.go.',
    '// Regenerate with: deno task gen:enums',
    '',
    sections.join('\n\n'),
    '',
  ].join('\n')
}

function parseGlossary(glossary: string): Map<string, Set<string>> {
  const interfaceMatch = /export interface GlossaryDimensionValueMap \{([\s\S]*?)\n\}/.exec(glossary)
  if (!interfaceMatch) fail('cannot find GlossaryDimensionValueMap in glossary.ts')

  const dimensions = new Set<string>()
  for (const line of interfaceMatch[1].split(/\r?\n/)) {
    if (line.trim() === '' || line.trimStart().startsWith('//')) continue
    const field = /^\s{2}([A-Za-z_][A-Za-z0-9_]*):\s+[A-Za-z_][A-Za-z0-9_]*\s*$/.exec(line)
    if (!field) fail(`cannot parse glossary dimension declaration: ${line.trim()}`)
    dimensions.add(field[1])
  }

  const valuesByDimension = new Map<string, Set<string>>()
  const tablePattern = /export const ([A-Za-z_][A-Za-z0-9_]*)Glossary: GlossaryTable<'([A-Za-z_][A-Za-z0-9_]*)'> = \{([\s\S]*?)\n\}/g
  for (const table of glossary.matchAll(tablePattern)) {
    const dimension = table[2]
    if (table[1] !== dimension) fail(`glossary table ${table[1]}Glossary declares dimension ${dimension}`)
    if (!dimensions.has(dimension)) fail(`glossary table references undeclared dimension ${dimension}`)
    if (valuesByDimension.has(dimension)) fail(`duplicate glossary table for ${dimension}`)

    const values = new Set<string>()
    for (const line of table[3].split(/\r?\n/)) {
      if (line.trim() === '' || line.trimStart().startsWith('//')) continue
      const entry = /^\s{2}([A-Za-z_][A-Za-z0-9_]*): entry\('([A-Za-z_][A-Za-z0-9_]*)', '([^']+)', '[^']+'\),\s*$/.exec(line)
      if (!entry) fail(`cannot parse ${dimension} glossary entry: ${line.trim()}`)
      if (entry[1] !== entry[3] || entry[2] !== dimension) {
        fail(`inconsistent ${dimension} glossary entry: ${line.trim()}`)
      }
      values.add(entry[1])
    }
    valuesByDimension.set(dimension, values)
  }

  for (const dimension of dimensions) {
    if (!valuesByDimension.has(dimension)) fail(`no glossary table found for dimension ${dimension}`)
  }
  return valuesByDimension
}

function checkGlossaryCoverage(enums: readonly GoEnum[], glossary: string): void {
  const glossaryValues = parseGlossary(glossary)
  const mappedDimensions = new Set(Object.values(GO_ENUM_TO_GLOSSARY_DIMENSION))
  const errors: string[] = []

  for (const dimension of glossaryValues.keys()) {
    if (!mappedDimensions.has(dimension) && !NON_DOMAIN_GLOSSARY_DIMENSIONS.has(dimension)) {
      errors.push(`glossary dimension ${dimension} is neither mapped nor allowlisted`)
    }
  }
  for (const dimension of NON_DOMAIN_GLOSSARY_DIMENSIONS) {
    if (!glossaryValues.has(dimension)) errors.push(`allowlisted glossary dimension ${dimension} does not exist`)
  }

  const enumByName = new Map(enums.map((goEnum) => [goEnum.name, goEnum]))
  for (const [enumName, dimension] of Object.entries(GO_ENUM_TO_GLOSSARY_DIMENSION)) {
    const goEnum = enumByName.get(enumName)
    if (!goEnum) {
      errors.push(`mapped Go enum ${enumName} does not exist`)
      continue
    }
    const actual = glossaryValues.get(dimension)
    if (!actual) {
      errors.push(`mapped glossary dimension ${dimension} does not exist`)
      continue
    }
    const expected = new Set(goEnum.values)
    const missing = goEnum.values.filter((value) => !actual.has(value))
    const extra = [...actual].filter((value) => !expected.has(value))
    if (missing.length > 0 || extra.length > 0) {
      errors.push(`${enumName} -> ${dimension}: missing [${missing.join(', ')}], extra [${extra.join(', ')}]`)
    }
  }

  if (errors.length > 0) fail(`glossary coverage failed:\n  ${errors.join('\n  ')}`)
}

function firstDifference(expected: string, actual: string): string {
  const expectedLines = expected.split('\n')
  const actualLines = actual.split('\n')
  const length = Math.max(expectedLines.length, actualLines.length)
  for (let index = 0; index < length; index++) {
    if (expectedLines[index] !== actualLines[index]) {
      return `first difference at line ${index + 1}:\n- ${actualLines[index] ?? '<missing>'}\n+ ${expectedLines[index] ?? '<missing>'}`
    }
  }
  return 'content differs'
}

const unknownArgs = Deno.args.filter((arg) => arg !== '--check')
if (unknownArgs.length > 0) fail(`unknown argument(s): ${unknownArgs.join(', ')}`)

const checkMode = Deno.args.includes('--check')
const enums = parseGoEnums(await Deno.readTextFile(GO_SOURCE_URL))
const generated = generate(enums)

if (checkMode) {
  checkGlossaryCoverage(enums, await Deno.readTextFile(GLOSSARY_URL))
  let existing: string
  try {
    existing = await Deno.readTextFile(OUTPUT_URL)
  } catch (error) {
    if (error instanceof Deno.errors.NotFound) fail('generated enums.ts is missing; run deno task gen:enums')
    throw error
  }
  if (existing !== generated) fail(`generated enums.ts is stale; run deno task gen:enums\n${firstDifference(generated, existing)}`)
  console.log(`Generated enums are current (${enums.length} Go enums); glossary coverage passed.`)
} else {
  let existing: string | undefined
  try {
    existing = await Deno.readTextFile(OUTPUT_URL)
  } catch (error) {
    if (!(error instanceof Deno.errors.NotFound)) throw error
  }
  if (existing === generated) {
    console.log(`Generated enums are already current (${enums.length} Go enums).`)
  } else {
    await Deno.mkdir(new URL('.', OUTPUT_URL), { recursive: true })
    await Deno.writeTextFile(OUTPUT_URL, generated)
    console.log(`Generated ${enums.length} Go enums in src/shared/api/generated/enums.ts.`)
  }
}
