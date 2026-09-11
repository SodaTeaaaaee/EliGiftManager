// Deno-native quality guardrail for frontend (redesign plan §6.3).
// Run via: deno task lint:guardrails  (== deno run --allow-read scripts/guardrails.ts)
//
// This is a standalone scanner, NOT an ESLint/deno-lint plugin, because
// `deno lint` does not process .vue files at all. It walks frontend/src,
// applies five structural rules, prints violations as file:line + rule id +
// snippet, and exits 1 if any violation was found.
//
// Rules:
//   no-hardcoded-text   Vue <template> text nodes / a small set of static
//                        user-visible attributes must go through t(...).
//   no-raw-enum         enum-suffixed fields must not be interpolated as
//                        raw text outside StatusBadge/StatusDot/t()/glossary.
//   no-restricted-import  src/pages/** must not import frontend/bindings
//                        (wails v3 generated bindings — runtime AND type
//                        imports alike, no type-only exemption; runtime calls
//                        go through src/shared/api/bridge.ts, types through
//                        the @/entities facade), must not import the
//                        deprecated wails v2 `wailsjs/` generated tree, and
//                        must not import Naive UI layout/feedback components
//                        directly (shared/ui).
//   no-missing-locale-key  every t('...') literal in src/.vue/.ts must
//                        resolve to a key in zh-CN.ts (en-US.ts is kept in
//                        lockstep via AppMessageSchema typing) or to a
//                        glossary.<dimension>.<value>.label|.description key
//                        generated from glossary.ts.
//   no-unused-export    every named export of a src .ts module must be
//                        imported somewhere in src (.vue SFCs, main.ts and
//                        barrel re-exports handled per exemption rules).
//   stale-generated-enums  the wails v3 generated TS enums (BlockReason,
//                        WorkState, incl. their synthetic $zero member)
//                        must match the *Values mirror arrays in
//                        src/shared/api/generated/enums.ts value-for-value.

const SRC_ROOT = "src";
const GLOSSARY_PATH = "src/shared/i18n/glossary.ts";

// The default-locale message bundle defines the canonical key set. Importing
// the module (instead of re-parsing its text) keeps this rule exact even when
// values are nested objects or template strings.
import { zhCN } from "../src/shared/i18n/locales/zh-CN.ts";

interface Violation {
  file: string;
  line: number;
  rule:
    | "no-hardcoded-text"
    | "no-raw-enum"
    | "no-restricted-import"
    | "no-missing-locale-key"
    | "no-unused-export"
    | "stale-generated-enums";
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
  // Named-clause identifiers (original exported name, alias resolved away).
  // A pure default import contributes its default identifier; namespace,
  // bare, dynamic and re-export-star forms bind no named exports.
  names: string[];
  index: number;
}

const NAME_PART = String.raw`[\w$]+`;
const NAMED_CLAUSE = String.raw`\{[^{}]*\}`;

/**
 * Extracts every import form Vite/vue-tsc can statically resolve: named,
 * default, namespace (`import * as ns`), mixed (`import d, { x }`),
 * type-only (statement- and element-level), bare side-effect
 * (`import '...'`), re-exports (`export { x } from`,
 * `export * (as ns)? from`) and dynamic `import('...')` with a literal.
 * Dynamic specifiers computed at runtime (`import(variable)`, template
 * literals with substitutions) are not statically checkable and stay out
 * of scope.
 */
function extractImportStatements(scriptContent: string): ImportStatement[] {
  const out: ImportStatement[] = [];

  // `import (type )?X(, (* as ns | { named }))? from 'spec'` and its
  // namespace / named-only variants.
  const clauseRe = new RegExp(
    String.raw`\bimport\s+(type\s+)?(?:\*(?:\s+as\s+(${NAME_PART}))?|(${NAME_PART})(?:\s*,\s*(?:\*(?:\s+as\s+(${NAME_PART}))?|(${NAMED_CLAUSE})))?|(${NAMED_CLAUSE}))\s+from\s+(['"])((?:\\.|(?!\7).)*)\7`,
    "g",
  );
  // Bare side-effect `import 'spec'`.
  const bareRe = /\bimport\s*(['"])([^'"\n]+)\1/g;
  // Dynamic `import('spec')`; the backtick form is matched too and filtered
  // below when it carries substitutions.
  const dynamicRe = /\bimport\s*\(\s*(['"`])((?:\\.|(?!\1)[^\\])*)\1/g;
  // Re-export `export (type )?(* (as ns)? | { named }) from 'spec'`.
  const reexportRe = new RegExp(
    String.raw`\bexport\s+(?:type\s+)?(?:\*(?:\s+as\s+(${NAME_PART}))?|(${NAMED_CLAUSE}))\s+from\s+(['"])((?:\\.|(?!\3).)*)\3`,
    "g",
  );

  const parseNamedClause = (clause: string): string[] =>
    clause
      .slice(1, -1)
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean)
      .map((s) => s.replace(/^type\s+/, "").split(/\s+as\s+/)[0].trim());

  let m: RegExpExecArray | null;
  clauseRe.lastIndex = 0;
  while ((m = clauseRe.exec(scriptContent)) !== null) {
    const isTypeOnly = Boolean(m[1]);
    const defaultName = m[3];
    const namedClause = m[5] ?? m[6];
    // Mixed statements expose only their named clause, so no-unused-export
    // never mistakes a default binding for an imported export name; a pure
    // default import keeps its identifier in `names` for the naive-ui
    // named-import check.
    const names = namedClause
      ? parseNamedClause(namedClause)
      : defaultName && !m[2] && !m[4]
      ? [defaultName]
      : [];
    out.push({
      raw: m[0],
      isTypeOnly,
      specifier: m[8],
      names,
      index: m.index,
    });
  }

  bareRe.lastIndex = 0;
  while ((m = bareRe.exec(scriptContent)) !== null) {
    out.push({
      raw: m[0],
      isTypeOnly: false,
      specifier: m[2],
      names: [],
      index: m.index,
    });
  }

  dynamicRe.lastIndex = 0;
  while ((m = dynamicRe.exec(scriptContent)) !== null) {
    if (m[1] === "`" && m[2].includes("${")) continue; // computed — skip
    out.push({
      raw: m[0],
      isTypeOnly: false,
      specifier: m[2],
      names: [],
      index: m.index,
    });
  }

  reexportRe.lastIndex = 0;
  while ((m = reexportRe.exec(scriptContent)) !== null) {
    out.push({
      raw: m[0],
      isTypeOnly: /^export\s+type\s/.test(m[0]),
      specifier: m[4],
      names: m[2] ? parseNamedClause(m[2]) : [],
      index: m.index,
    });
  }

  return out;
}

/**
 * True when a raw import specifier, resolved from the importing file under
 * src/**, lands in frontend/bindings/. Covered specifier shapes — the ones
 * Vite/vue-tsc can physically resolve from a page:
 *   - relative climbs:            `../../bindings/...`
 *   - Vite root-absolute paths:   `/bindings/...` (the leading `/` resolves
 *     from the frontend project root)
 *   - `@/` escapes:               `@` aliases to `src`, so `@/../bindings`
 *     escapes to the project root
 * Segment comparison is case-insensitive (Windows filesystems resolve
 * `Bindings` like `bindings`), and climbs that leave src/ and re-enter
 * through the frontend root (`@/../../frontend/bindings/...`,
 * `../../../../frontend/bindings/...`) normalize to a
 * `["frontend", "bindings", ...]` stack and are caught as well.
 * Bare specifiers cannot reach it (bindings is not an npm package and has
 * no alias), so they are rejected outright.
 */
function resolvesUnderBindings(specifier: string, relFile: string): boolean {
  let path: string;
  if (specifier.startsWith("/")) {
    path = specifier;
  } else if (specifier.startsWith("./") || specifier.startsWith("../")) {
    path = `${relFile.slice(0, relFile.lastIndexOf("/"))}/${specifier}`;
  } else if (specifier.startsWith("@/")) {
    path = specifier.replace(/^@\//, "src/");
  } else {
    return false;
  }
  const stack: string[] = [];
  for (const part of path.split("/")) {
    if (part === "" || part === ".") continue;
    if (part === "..") {
      stack.pop();
      continue;
    }
    stack.push(part);
  }
  const first = stack[0]?.toLowerCase();
  if (first === "bindings") return true;
  // Re-entry through the frontend root after climbing past src/.
  return first === "frontend" && stack[1]?.toLowerCase() === "bindings";
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

    if (resolvesUnderBindings(imp.specifier, relFile)) {
      violations.push({
        file: relFile,
        line,
        rule: "no-restricted-import",
        message:
          `page imports "${imp.specifier}" directly — route runtime calls through src/shared/api/bridge.ts and types through @/entities (frontend/bindings is banned in src/pages, no type-only exemption)`,
        snippet: snippetOf(imp.raw),
      });
    }

    // wails v2 generated tree — deprecated by the wails v3 migration.
    // Kept as a tripwire even after frontend/wailsjs/ is deleted.
    if (imp.specifier.toLowerCase().includes("wailsjs")) {
      violations.push({
        file: relFile,
        line,
        rule: "no-restricted-import",
        message:
          `page imports "${imp.specifier}" — wailsjs (wails v2) generated artifacts are deprecated; route runtime calls through src/shared/api/bridge.ts and types through @/entities`,
        snippet: snippetOf(imp.raw),
      });
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
// Rule: no-missing-locale-key
// ---------------------------------------------------------------------------

/** Path prefixes/files whose own t() usage defines keys instead of consuming them. */
const LOCALE_SCAN_EXEMPT = [
  "src/shared/i18n/locales/",
  "src/shared/i18n/glossary.ts",
  "src/shared/api/generated/",
];

function flattenMessageKeys(
  node: Record<string, unknown>,
  prefix = "",
): Set<string> {
  const keys = new Set<string>();
  for (const [name, value] of Object.entries(node)) {
    const key = prefix ? `${prefix}.${name}` : name;
    if (value && typeof value === "object") {
      for (const k of flattenMessageKeys(value as Record<string, unknown>, key)) {
        keys.add(k);
      }
    } else {
      keys.add(key);
    }
  }
  return keys;
}

/** glossary.<dimension>.<value>.label|.description keys, parsed like gen-enums.ts. */
function collectGlossaryDynamicKeys(glossarySource: string): Set<string> {
  const keys = new Set<string>();
  const entryRe = /entry\('([A-Za-z_]+)',\s*'([A-Za-z_]+)',/g;
  let m: RegExpExecArray | null;
  while ((m = entryRe.exec(glossarySource)) !== null) {
    keys.add(`glossary.${m[1]}.${m[2]}.label`);
    keys.add(`glossary.${m[1]}.${m[2]}.description`);
  }
  return keys;
}

/** Matches a t(...) call whose first argument is a plain string literal. */
const T_LITERAL_CALL_RE = /\bt\(\s*(['"])((?:\\.|(?!\1).)*)\1/g;

/**
 * A statically checkable message key: full dotted path of word characters
 * (`a.b.c`). Anything else (a truncated concatenation fragment like `a.`, a
 * key with interpolation markers, a bare single word) is treated as a dynamic
 * key and skipped by no-missing-locale-key.
 */
const CHECKABLE_KEY_RE = /^\w+(\.\w+)+$/;

/** Masks `<!-- ... -->` HTML comments with spaces (length-preserving). */
function maskHtmlComments(content: string): string {
  return content.replace(/<!--[\s\S]*?-->/g, (m) => " ".repeat(m.length));
}

/**
 * Masks JS line/block comments with spaces so t('...') literals that only
 * appear inside comments cannot produce false positives for
 * no-missing-locale-key. Length-preserving, so the offsets used for
 * file:line reporting stay valid. Comment markers inside string literals
 * ("https://..." and friends) are tracked and left untouched. Only applied
 * to script blocks and .ts sources — never to <template> text, where a stray
 * apostrophe in copy would derail the quote tracking.
 */
function maskJsComments(content: string): string {
  const chars = content.split("");
  const n = chars.length;
  let i = 0;
  const blankUntil = (stop: number): void => {
    for (let k = i; k < stop; k++) {
      if (chars[k] !== "\n") chars[k] = " ";
    }
  };
  while (i < n) {
    const c = chars[i];
    if (c === '"' || c === "'" || c === "`") {
      i++;
      while (i < n && chars[i] !== c) {
        if (chars[i] === "\\") i++; // skip the escaped character
        i++;
      }
      i++;
      continue;
    }
    if (c === "/" && chars[i + 1] === "/") {
      while (i < n && chars[i] !== "\n") {
        chars[i] = " ";
        i++;
      }
      continue; // keep the newline itself
    }
    if (c === "/" && chars[i + 1] === "*") {
      const end = content.indexOf("*/", i + 2);
      const stop = end === -1 ? n : end + 2;
      blankUntil(stop);
      i = stop;
      continue;
    }
    i++;
  }
  return chars.join("");
}

function checkLocaleKeyLiterals(
  content: string,
  contentStartOffset: number,
  fileContent: string,
  relFile: string,
  knownKeys: Set<string>,
  maskComments: (s: string) => string,
): void {
  const scanned = maskComments(content);
  let m: RegExpExecArray | null;
  T_LITERAL_CALL_RE.lastIndex = 0;
  while ((m = T_LITERAL_CALL_RE.exec(scanned)) !== null) {
    const key = m[2];
    // Dynamic/partial keys (template-literal composition happens elsewhere)
    // cannot be statically resolved and are skipped.
    if (key.includes("${")) continue;
    // Concatenation defense: a fragment that is not a full dotted shape
    // (e.g. the `'designLab.'` half of `t('designLab.' + x)`) is dynamic.
    if (!CHECKABLE_KEY_RE.test(key)) continue;
    if (knownKeys.has(key)) continue;
    const line = indexToLine(fileContent, contentStartOffset + m.index);
    violations.push({
      file: relFile,
      line,
      rule: "no-missing-locale-key",
      message:
        `t("${key}") does not exist in zh-CN.ts — add the key to both locale bundles (en-US.ts is AppMessageSchema-typed to stay in sync)`,
      snippet: snippetOf(m[0]),
    });
  }
}

function scanLocaleKeys(files: string[], knownKeys: Set<string>): void {
  for (const file of files) {
    if (LOCALE_SCAN_EXEMPT.some((p) => file.startsWith(p) || file === p)) {
      continue;
    }
    const isVue = file.endsWith(".vue");
    if (!isVue && !file.endsWith(".ts")) continue;

    const fileContent = Deno.readTextFileSync(file);
    if (isVue) {
      const template = extractTemplateBlock(fileContent);
      if (template) {
        checkLocaleKeyLiterals(
          template.content,
          template.startOffset,
          fileContent,
          file,
          knownKeys,
          maskHtmlComments,
        );
      }
      for (const script of extractScriptBlocks(fileContent)) {
        checkLocaleKeyLiterals(
          script.content,
          script.startOffset,
          fileContent,
          file,
          knownKeys,
          maskJsComments,
        );
      }
    } else {
      checkLocaleKeyLiterals(
        fileContent,
        0,
        fileContent,
        file,
        knownKeys,
        maskJsComments,
      );
    }
  }
}

// ---------------------------------------------------------------------------
// Rule: no-unused-export
// ---------------------------------------------------------------------------

interface ExportDefinition {
  name: string;
  index: number;
}

/** Files that never count as violation sites for unused exports. */
const UNUSED_EXPORT_EXEMPT_FILES = new Set([
  // Generated by gen-enums.ts from the Go enums; its companion `*Values`
  // arrays exist for lockstep tooling, not for import.
  "src/shared/api/generated/enums.ts",
  // The glossary tables and value types are the module's structural API:
  // gen-enums.ts parses the `export const ...Glossary` tables and the
  // aggregate `glossaryTables` consumes them in-file.
  "src/shared/i18n/glossary.ts",
  // Deliberate hand-mirrors of backend DTOs kept for upcoming waves
  // (e.g. PlatformIdentity, ProductBundleComponent) even before pages
  // consume them.
  "src/entities/models.ts",
]);

function isUnusedExportExempt(relFile: string): boolean {
  if (relFile.endsWith(".d.ts")) return true;
  if (relFile === "src/main.ts") return true;
  return UNUSED_EXPORT_EXEMPT_FILES.has(relFile);
}

const EXPORT_DECL_RE =
  /\bexport\s+(?:async\s+)?(?:const|let|var|function\*?|class|interface|type|enum)\s+([A-Za-z_$][\w$]*)/g;

const EXPORT_CLAUSE_RE = /\bexport\s*\{([^}]*)\}/g;

function collectExportDefinitions(
  scriptContent: string,
): ExportDefinition[] {
  const defs: ExportDefinition[] = [];
  let m: RegExpExecArray | null;

  EXPORT_DECL_RE.lastIndex = 0;
  while ((m = EXPORT_DECL_RE.exec(scriptContent)) !== null) {
    defs.push({ name: m[1], index: m.index });
  }

  EXPORT_CLAUSE_RE.lastIndex = 0;
  while ((m = EXPORT_CLAUSE_RE.exec(scriptContent)) !== null) {
    for (const part of m[1].split(",")) {
      const item = part
        .trim()
        .replace(/^type\s+/, "")
        .split(/\s+as\s+/)
        .filter(Boolean);
      if (item.length === 0) continue;
      // `export { x as y }` exports y; `export { default }` is not named.
      const exportedName = item.length > 1 ? item[item.length - 1] : item[0];
      if (!exportedName || exportedName === "default") continue;
      defs.push({
        name: exportedName,
        index: m.index + (m[1].indexOf(part) >= 0 ? m[1].indexOf(part) : 0),
      });
    }
  }
  return defs;
}

/** Names referenced by named import clauses anywhere in src (alias-resolved to the exporter's name). */
function collectImportedNames(
  scriptContent: string,
  importedNames: Set<string>,
): void {
  for (const imp of extractImportStatements(scriptContent)) {
    // Only named clauses reference exported symbol names; default imports
    // (`import x from ...`) and namespace imports (`import * as ns`) do not.
    if (!imp.raw.match(/import\s+(type\s+)?\{/)) continue;
    for (const name of imp.names) importedNames.add(name);
  }
}

function scanUnusedExports(files: string[]): void {
  const importedNames = new Set<string>();
  const definitions: { file: string; def: ExportDefinition }[] = [];

  for (const file of files) {
    const isVue = file.endsWith(".vue");
    const isTs = file.endsWith(".ts");
    if (!isVue && !isTs) continue;

    const fileContent = Deno.readTextFileSync(file);

    // Every file contributes import references — .vue script blocks,
    // main.ts and *.test.ts included. .vue templates cannot import.
    if (isVue) {
      for (const script of extractScriptBlocks(fileContent)) {
        collectImportedNames(script.content, importedNames);
      }
      // .vue SFCs are exempt as definition sites: their default export is
      // referenced by filename (router lazy imports, component registration).
      continue;
    }

    collectImportedNames(fileContent, importedNames);

    // Definition sites to report: .ts modules except d.ts, main.ts and
    // *.test.ts (leaf consumers); their imports still count above.
    if (isUnusedExportExempt(file) || file.endsWith(".test.ts")) continue;
    for (const def of collectExportDefinitions(fileContent)) {
      definitions.push({ file, def });
    }
  }

  for (const { file, def } of definitions) {
    // A name counts as alive when ANY import clause anywhere references it.
    // This includes index.ts barrel re-exports: the barrel export lives iff
    // some file imports that name (from the barrel or elsewhere).
    if (importedNames.has(def.name)) continue;
    // Same-file usage also keeps an export alive: a name that appears in its
    // own file beyond the export declaration itself (another type reference,
    // an internal call, a bottom re-export clause) is live code even when no
    // other module imports it yet.
    const fileContent = Deno.readTextFileSync(file);
    const occurrenceRe = new RegExp(
      `(^|[^\\w$])${def.name.replace(/\$/g, "\\$")}(?![\\w$])`,
      "g",
    );
    let occurrences = 0;
    while (occurrenceRe.exec(fileContent) !== null) occurrences++;
    if (occurrences > 1) continue;
    const line = indexToLine(fileContent, def.index);
    violations.push({
      file,
      line,
      rule: "no-unused-export",
      message:
        `exported "${def.name}" is never imported anywhere in src and unused in its own module — remove the export or its dead module`,
      snippet: `export ... ${def.name}`,
    });
  }
}

// ---------------------------------------------------------------------------
// Rule: stale-generated-enums — bindings enum domains vs generated/enums.ts
// ---------------------------------------------------------------------------

// The wails v3 generator emits TS enums for the Go enums (BlockReason,
// WorkState, …), each carrying a synthetic `$zero` member for the Go zero
// value. gen-enums.ts mirrors the same Go enums into
// src/shared/api/generated/enums.ts as `*Values` arrays + string-literal
// unions — the form the UI actually consumes (pages compare raw literals,
// see the @/entities facade header). The two representations must stay
// value-for-value identical, order included ($zero excluded: it is the Go
// zero marker, never a real wire value), otherwise the UI keeps compiling
// against enum members the bindings can no longer deliver, or vice versa.
// Only enums that have a generated mirror participate; when gen-enums.ts
// grows a new one, add it to CROSS_CHECKED_ENUMS.
const BINDINGS_DOMAIN_MODELS_PATH =
  "bindings/github.com/SodaTeaaaaee/EliGiftManager/internal/domain/models.ts";
const GENERATED_ENUMS_PATH = "src/shared/api/generated/enums.ts";
const CROSS_CHECKED_ENUMS = ["BlockReason", "WorkState"] as const;

interface ParsedEnum {
  name: string;
  /** String-valued members in declaration order ($zero kept). */
  members: { name: string; value: string }[];
  line: number;
}

/** Parses `export enum X { A = "a", ... }` blocks out of a bindings module. */
function parseExportedEnums(source: string): ParsedEnum[] {
  const enums: ParsedEnum[] = [];
  const enumRe = /\bexport\s+enum\s+([A-Za-z_$][\w$]*)\s*\{/g;
  let m: RegExpExecArray | null;
  while ((m = enumRe.exec(source)) !== null) {
    const bodyStart = enumRe.lastIndex;
    const bodyEnd = source.indexOf("}", bodyStart);
    const body = bodyEnd === -1
      ? source.slice(bodyStart)
      : source.slice(bodyStart, bodyEnd);
    const memberRe = /([A-Za-z_$][\w$]*)\s*=\s*"((?:[^"\\]|\\.)*)"/g;
    const members: { name: string; value: string }[] = [];
    let mm: RegExpExecArray | null;
    while ((mm = memberRe.exec(body)) !== null) {
      members.push({ name: mm[1], value: mm[2] });
    }
    enums.push({ name: m[1], members, line: indexToLine(source, m.index) });
  }
  return enums;
}

/** Parses the `export const <lowerFirst(name)>Values = [ 'a', 'b' ] as const` mirror. */
function parseGeneratedValues(
  source: string,
  enumName: string,
): { arrayName: string; values: string[] } | null {
  const arrayName = enumName[0].toLowerCase() + enumName.slice(1) + "Values";
  const re = new RegExp(
    `export\\s+const\\s+${arrayName}\\s*=\\s*\\[([^\\]]*)\\]`,
  );
  const m = re.exec(source);
  if (!m) return null;
  const values: string[] = [];
  const strRe = /'([^']*)'/g;
  let mm: RegExpExecArray | null;
  while ((mm = strRe.exec(m[1])) !== null) values.push(mm[1]);
  return { arrayName, values };
}

function checkGeneratedEnumDomains(): void {
  const bindingsSource = Deno.readTextFileSync(BINDINGS_DOMAIN_MODELS_PATH);
  const generatedSource = Deno.readTextFileSync(GENERATED_ENUMS_PATH);
  const enumsByName = new Map(
    parseExportedEnums(bindingsSource).map((e) => [e.name, e]),
  );

  for (const enumName of CROSS_CHECKED_ENUMS) {
    const parsed = enumsByName.get(enumName);
    if (!parsed) {
      violations.push({
        file: BINDINGS_DOMAIN_MODELS_PATH,
        line: 1,
        rule: "stale-generated-enums",
        message:
          `expected generated TS enum ${enumName} not found — rerun wails3 generate bindings`,
        snippet: BINDINGS_DOMAIN_MODELS_PATH,
      });
      continue;
    }
    const bindingValues = parsed.members
      .filter((mem) => mem.name !== "$zero")
      .map((mem) => mem.value);
    const mirror = parseGeneratedValues(generatedSource, enumName);
    if (!mirror) {
      violations.push({
        file: GENERATED_ENUMS_PATH,
        line: 1,
        rule: "stale-generated-enums",
        message:
          `no ${enumName[0].toLowerCase() + enumName.slice(1)}Values array found for bindings enum ${enumName} — rerun deno task gen:enums`,
        snippet: GENERATED_ENUMS_PATH,
      });
      continue;
    }
    if (bindingValues.join("\n") !== mirror.values.join("\n")) {
      violations.push({
        file: BINDINGS_DOMAIN_MODELS_PATH,
        line: parsed.line,
        rule: "stale-generated-enums",
        message:
          `enum ${enumName} domain out of sync with ${GENERATED_ENUMS_PATH} ${mirror.arrayName}: bindings declare [${bindingValues.join(", ")}] but the mirror lists [${mirror.values.join(", ")}] — rerun wails3 generate bindings and/or deno task gen:enums`,
        snippet: `export enum ${enumName} { ... }`,
      });
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

  // Rule 4: every t('...') literal must resolve to a known locale key.
  // zh-CN.ts is the single source of truth; en-US.ts is typed as
  // AppMessageSchema = typeof zhCN so its key set cannot drift.
  const knownKeys = flattenMessageKeys(zhCN as Record<string, unknown>);
  for (const key of collectGlossaryDynamicKeys(
    Deno.readTextFileSync(GLOSSARY_PATH),
  )) {
    knownKeys.add(key);
  }
  scanLocaleKeys(files, knownKeys);

  // Rule 5: named exports must be imported somewhere in src.
  scanUnusedExports(files);

  // Rule 6: generated enum domains must match the bindings value-for-value.
  checkGeneratedEnumDomains();

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
