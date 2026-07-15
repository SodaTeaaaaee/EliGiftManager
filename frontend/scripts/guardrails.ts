// Deno-native quality guardrail for frontend (redesign plan §6.3).
// Run via: deno task lint:guardrails  (== deno run --allow-read scripts/guardrails.ts)
//
// This is a standalone scanner, NOT an ESLint/deno-lint plugin, because
// `deno lint` does not process .vue files at all. It walks frontend/src,
// applies three structural rules, prints violations as file:line + rule id +
// snippet, and exits 1 if any violation was found.
//
// Rules:
//   no-hardcoded-text   Vue <template> text nodes / a small set of static
//                        user-visible attributes must go through t(...).
//   no-raw-enum         enum-suffixed fields must not be interpolated as
//                        raw text outside StatusBadge/StatusDot/t()/glossary.
//   no-restricted-import  src/pages/** must not import wailsjs directly
//                        (except `import type { dto } from '.../wailsjs/go/models'`)
//                        and must not import Naive UI layout/feedback
//                        components directly (must go through shared/ui).

const SRC_ROOT = "src";

interface Violation {
  file: string;
  line: number;
  rule: "no-hardcoded-text" | "no-raw-enum" | "no-restricted-import";
  message: string;
  snippet: string;
}

const violations: Violation[] = [];

// ---------------------------------------------------------------------------
// File discovery
// ---------------------------------------------------------------------------

const SKIP_DIR_NAMES = new Set(["node_modules", "dist", "wailsjs", ".git"]);

function walk(dir: string): string[] {
  const out: string[] = [];
  let entries: Deno.DirEntry[];
  try {
    entries = [...Deno.readDirSync(dir)];
  } catch {
    return out;
  }
  for (const entry of entries) {
    if (SKIP_DIR_NAMES.has(entry.name)) continue;
    const full = `${dir}/${entry.name}`;
    if (entry.isDirectory) {
      out.push(...walk(full));
    } else if (entry.isFile) {
      out.push(full);
    }
  }
  return out;
}

function toPosix(path: string): string {
  return path.replace(/\\/g, "/");
}

function indexToLine(content: string, index: number): number {
  let line = 1;
  for (let i = 0; i < index && i < content.length; i++) {
    if (content[i] === "\n") line++;
  }
  return line;
}

function snippetOf(text: string, maxLen = 100): string {
  const oneLine = text.replace(/\s+/g, " ").trim();
  return oneLine.length > maxLen ? oneLine.slice(0, maxLen) + "…" : oneLine;
}

// ---------------------------------------------------------------------------
// <template> block extraction (depth-aware, handles nested `<template #x>`
// slot tags used for named slots inside the SFC root template).
// ---------------------------------------------------------------------------

interface TemplateBlock {
  content: string;
  /** absolute char offset of `content[0]` within the full file. */
  startOffset: number;
}

// Matches a run of tag "innards" that safely skips over `>` (and `<`)
// characters that appear *inside* quoted attribute values (e.g.
// `v-if="x.length > 0"`), so the tag boundary is only the real, unquoted `>`.
const TAG_INNARDS = `(?:"[^"]*"|'[^']*'|[^'">])*`;

function extractTemplateBlock(fileContent: string): TemplateBlock | null {
  const openTagRe = new RegExp(`<template(?:\\s${TAG_INNARDS})?>`, "i");
  const startMatch = openTagRe.exec(fileContent);
  if (!startMatch) return null;
  const contentStart = startMatch.index + startMatch[0].length;

  const tagRe = new RegExp(
    `<template(?:\\s${TAG_INNARDS})?>|<\\/template>`,
    "gi",
  );
  tagRe.lastIndex = contentStart;
  let depth = 1;
  let m: RegExpExecArray | null;
  while ((m = tagRe.exec(fileContent)) !== null) {
    if (m[0].toLowerCase() === "</template>") {
      depth--;
      if (depth === 0) {
        return {
          content: fileContent.slice(contentStart, m.index),
          startOffset: contentStart,
        };
      }
    } else {
      depth++;
    }
  }
  // Unterminated (malformed file) — take the rest, best-effort.
  return {
    content: fileContent.slice(contentStart),
    startOffset: contentStart,
  };
}

// ---------------------------------------------------------------------------
// <script> block extraction (no nesting concern — SFCs never nest <script>).
// ---------------------------------------------------------------------------

interface ScriptBlock {
  content: string;
  /** absolute char offset of `content[0]` within the full file. */
  startOffset: number;
}

function extractScriptBlocks(fileContent: string): ScriptBlock[] {
  const blocks: ScriptBlock[] = [];
  const scriptRe = /<script(\s[^>]*)?>([\s\S]*?)<\/script>/gi;
  let m: RegExpExecArray | null;
  while ((m = scriptRe.exec(fileContent)) !== null) {
    const contentStart = m.index + m[0].length - m[2].length -
      "</script>".length;
    blocks.push({ content: m[2], startOffset: contentStart });
  }
  return blocks;
}

// ---------------------------------------------------------------------------
// Rule: no-hardcoded-text / no-raw-enum — template tokenizer
// ---------------------------------------------------------------------------

const ALLOWLISTED_TEXT_ATTRS = new Set([
  "title",
  "placeholder",
  "label",
  "message",
  "aria-label",
  "content",
  "description",
]);

const VOID_ELEMENTS = new Set([
  "area",
  "base",
  "br",
  "col",
  "embed",
  "hr",
  "img",
  "input",
  "link",
  "meta",
  "param",
  "source",
  "track",
  "wbr",
]);

const ENUM_SUFFIX_RE =
  /(^|[^A-Za-z0-9_$])[A-Za-z_$][A-Za-z0-9_$]*?(?:State|Status|Stage|Mode|Type|Disposition|Reason|Policy|Kind)(?![A-Za-z0-9_$])/;

/** Any Unicode letter (covers CJK + Latin). */
const HAS_LETTER_RE = /\p{L}/u;

function getTagName(tagToken: string): string {
  const m = /^<\/?([A-Za-z][\w.:-]*)/.exec(tagToken);
  return m ? m[1] : "";
}

function isClosingTag(tagToken: string): boolean {
  return /^<\//.test(tagToken);
}

function isSelfClosingTag(tagToken: string): boolean {
  if (/\/\s*>$/.test(tagToken)) return true;
  const name = getTagName(tagToken).toLowerCase();
  return VOID_ELEMENTS.has(name);
}

/** True if the tag token carries a STATIC (non-`:`/non-`v-bind:`) aria-hidden="true" attribute. */
function hasStaticAriaHiddenTrue(tagToken: string): boolean {
  return /(^|\s)aria-hidden\s*=\s*"true"/.test(tagToken);
}

/** Checks the tag's own static allowlisted attributes for hardcoded user-visible text. */
function checkTagAttrs(
  tagToken: string,
  tagStartIndex: number,
  fileContent: string,
  relFile: string,
): void {
  const attrRe = /(^|\s)([A-Za-z-]+)\s*=\s*"([^"]*)"/g;
  let m: RegExpExecArray | null;
  while ((m = attrRe.exec(tagToken)) !== null) {
    const attrName = m[2].toLowerCase();
    if (!ALLOWLISTED_TEXT_ATTRS.has(attrName)) continue;
    const value = m[3];
    if (HAS_LETTER_RE.test(value)) {
      const line = indexToLine(
        fileContent,
        tagStartIndex + m.index + m[1].length,
      );
      violations.push({
        file: relFile,
        line,
        rule: "no-hardcoded-text",
        message:
          `static attribute "${attrName}" has a hardcoded user-visible value — use :${attrName}="t(...)" instead`,
        snippet: snippetOf(m[0]),
      });
    }
  }
}

/** Rule no-raw-enum, applied to the inner expression of one `{{ ... }}` interpolation. */
function checkInterpolationForRawEnum(
  expr: string,
  exprStartIndex: number,
  fileContent: string,
  relFile: string,
): void {
  const trimmed = expr.trim();
  if (trimmed.length === 0) return;
  if (/\bt\(/.test(trimmed)) return; // routed through i18n / a t()-based resolver — trust it.

  const hasParens = /\(/.test(trimmed);
  const isTernary = /\?(?!\.)[\s\S]*:/.test(trimmed); // `?` not followed by `.` (excludes `?.` optional chaining)

  if (!hasParens && ENUM_SUFFIX_RE.test(trimmed)) {
    violations.push({
      file: relFile,
      line: indexToLine(fileContent, exprStartIndex),
      rule: "no-raw-enum",
      message:
        "enum-suffixed field interpolated as raw text — render via <StatusBadge>/<StatusDot> or a glossary/t() resolver",
      snippet: snippetOf(trimmed),
    });
    return;
  }

  if (isTernary && ENUM_SUFFIX_RE.test(trimmed)) {
    violations.push({
      file: relFile,
      line: indexToLine(fileContent, exprStartIndex),
      rule: "no-raw-enum",
      message:
        "ternary keyed off an enum-suffixed field renders raw text outside StatusBadge/StatusDot/glossary",
      snippet: snippetOf(trimmed),
    });
  }
}

function scanTemplate(
  block: TemplateBlock,
  fileContent: string,
  relFile: string,
): void {
  // Strip comments first so `>`/`<` inside them can't confuse the tokenizer.
  const content = block.content.replace(
    /<!--[\s\S]*?-->/g,
    (m) => " ".repeat(m.length),
  );

  const tokenRe = new RegExp(`<${TAG_INNARDS}>|[^<]+`, "g");
  let m: RegExpExecArray | null;
  // stack of whether each currently-open element is in "aria-hidden decorative"
  // mode (own attribute OR inherited from an ancestor).
  const ariaHiddenStack: boolean[] = [];

  while ((m = tokenRe.exec(content)) !== null) {
    const token = m[0];
    const tokenStart = block.startOffset + m.index;

    if (token.startsWith("<")) {
      if (isClosingTag(token)) {
        if (ariaHiddenStack.length > 0) ariaHiddenStack.pop();
        continue;
      }
      checkTagAttrs(token, tokenStart, fileContent, relFile);
      const ownAriaHidden = hasStaticAriaHiddenTrue(token);
      const parentAriaHidden = ariaHiddenStack.length > 0
        ? ariaHiddenStack[ariaHiddenStack.length - 1]
        : false;
      const effectiveAriaHidden = ownAriaHidden || parentAriaHidden;
      if (!isSelfClosingTag(token)) {
        ariaHiddenStack.push(effectiveAriaHidden);
      }
      continue;
    }

    // TEXT token.
    const inSkip = ariaHiddenStack.length > 0
      ? ariaHiddenStack[ariaHiddenStack.length - 1]
      : false;

    // Always scan mustache interpolations for rule no-raw-enum, even inside
    // aria-hidden regions (an enum leaking into a hidden node is still a bug
    // upstream, and no such case is expected in practice).
    const mustacheRe = /\{\{([\s\S]*?)\}\}/g;
    let mm: RegExpExecArray | null;
    while ((mm = mustacheRe.exec(token)) !== null) {
      const exprStart = tokenStart + mm.index + 2; // skip "{{"
      checkInterpolationForRawEnum(mm[1], exprStart, fileContent, relFile);
    }
    const sansInterpolation = token.replace(mustacheRe, "");

    if (inSkip) continue;
    if (HAS_LETTER_RE.test(sansInterpolation)) {
      violations.push({
        file: relFile,
        line: indexToLine(fileContent, tokenStart),
        rule: "no-hardcoded-text",
        message: "hardcoded text node — wrap in t(...) or a glossary lookup",
        snippet: snippetOf(token),
      });
    }
  }
}

// ---------------------------------------------------------------------------
// Rule: no-restricted-import (src/pages/** only)
// ---------------------------------------------------------------------------

// NModal and NDrawer are RETAINED Naive UI kernels per redesign plan §4.3
// ("保留 Naive UI" / retain list explicitly includes "NModal/NDrawer 内核")
// and are therefore intentionally NOT in this banned set. Only components
// with a self-built replacement in src/shared/ui are banned here:
// FeedbackSystem/ToastViewport for messages/notifications/dialogs,
// TopProgressBar for loading bars, AppShell/SideNav for layout.
const BANNED_NAIVE_EXACT = new Set([
  "NMessage",
  "useMessage",
  "NNotification",
  "useNotification",
  "NDialog",
  "useDialog",
  "NLoadingBar",
  "useLoadingBar",
  "NLayout",
  "NLayoutSider",
  "NLayoutHeader",
  "NLayoutContent",
  "NLayoutFooter",
]);

function isBannedNaiveName(name: string): boolean {
  return BANNED_NAIVE_EXACT.has(name);
}

interface ImportStatement {
  raw: string;
  isTypeOnly: boolean;
  specifier: string;
  names: string[]; // named-import identifiers (original exported name, alias resolved away)
  index: number;
}

function extractImportStatements(scriptContent: string): ImportStatement[] {
  const out: ImportStatement[] = [];
  const importRe =
    /import\s+(type\s+)?(\{[\s\S]*?\}|[\w$]+)\s+from\s+['"]([^'"]+)['"]/g;
  let m: RegExpExecArray | null;
  while ((m = importRe.exec(scriptContent)) !== null) {
    const isTypeOnly = Boolean(m[1]);
    const clause = m[2];
    const specifier = m[3];
    let names: string[] = [];
    if (clause.startsWith("{")) {
      names = clause
        .slice(1, -1)
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean)
        .map((s) => s.replace(/^type\s+/, "").split(/\s+as\s+/)[0].trim());
    } else {
      names = [clause.trim()];
    }
    out.push({ raw: m[0], isTypeOnly, specifier, names, index: m.index });
  }
  return out;
}

function scanImportsInPages(
  scriptContent: string,
  scriptStartOffset: number,
  fileContent: string,
  relFile: string,
): void {
  for (const imp of extractImportStatements(scriptContent)) {
    const absIndex = scriptStartOffset + imp.index;
    const line = indexToLine(fileContent, absIndex);

    if (/wailsjs/i.test(imp.specifier)) {
      const isModelsPath = /wailsjs\/go\/models$/.test(imp.specifier);
      const isExempt = imp.isTypeOnly && isModelsPath;
      if (!isExempt) {
        violations.push({
          file: relFile,
          line,
          rule: "no-restricted-import",
          message:
            `page imports "${imp.specifier}" directly — route runtime calls through src/shared/api/bridge.ts (only \`import type { dto } from '.../wailsjs/go/models'\` is exempt)`,
          snippet: snippetOf(imp.raw),
        });
      }
    }

    if (imp.specifier === "naive-ui") {
      const banned = imp.names.filter(isBannedNaiveName);
      if (banned.length > 0) {
        violations.push({
          file: relFile,
          line,
          rule: "no-restricted-import",
          message: `page imports Naive layout/feedback component(s) directly: ${
            banned.join(", ")
          } — use the src/shared/ui wrapper instead`,
          snippet: snippetOf(imp.raw),
        });
      }
    }
  }
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

function isUnderPages(relFile: string): boolean {
  return relFile.startsWith("src/pages/");
}

function main(): void {
  const files = walk(SRC_ROOT).map(toPosix);

  for (const file of files) {
    const relFile = file; // already relative (walk starts at "src")
    const isVue = file.endsWith(".vue");
    const isTs = file.endsWith(".ts") || file.endsWith(".tsx");
    if (!isVue && !isTs) continue;

    const fileContent = Deno.readTextFileSync(file);

    // Rules 1 & 2: template content, .vue files only.
    if (isVue) {
      const block = extractTemplateBlock(fileContent);
      if (block) scanTemplate(block, fileContent, relFile);
    }

    // Rule 3: src/pages/** only. (src/shared/api/ — the bridge/health wrapper
    // layer — is never under src/pages/**, so it is naturally out of scope.)
    if (isUnderPages(relFile)) {
      if (isVue) {
        for (const scriptBlock of extractScriptBlocks(fileContent)) {
          scanImportsInPages(
            scriptBlock.content,
            scriptBlock.startOffset,
            fileContent,
            relFile,
          );
        }
      } else {
        scanImportsInPages(fileContent, 0, fileContent, relFile);
      }
    }
  }

  if (violations.length === 0) {
    console.log("guardrails: no violations found.");
    return;
  }

  violations.sort((
    a,
    b,
  ) => (a.file === b.file ? a.line - b.line : a.file.localeCompare(b.file)));

  for (const v of violations) {
    console.log(`${v.file}:${v.line}  [${v.rule}]  ${v.message}`);
    console.log(`    ${v.snippet}`);
  }
  console.log(`\nguardrails: ${violations.length} violation(s) found.`);
  Deno.exit(1);
}

main();
