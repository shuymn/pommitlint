import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, join } from "node:path";

import config from "@commitlint/config-conventional";
import lint from "@commitlint/lint";
import isIgnored from "@commitlint/is-ignored";
import createPreset from "conventional-changelog-conventionalcommits";

interface CorpusEntry {
  id: string;
  message: string;
  tags: string[];
}

interface FindingSummary {
  rule: string;
  level: string;
}

interface LintResult {
  id: string;
  valid: boolean;
  ignored: boolean;
  findings: FindingSummary[];
}

const scriptDir = dirname(fileURLToPath(import.meta.url));
const corpusPath = join(scriptDir, "corpus.json");
const corpus: CorpusEntry[] = JSON.parse(readFileSync(corpusPath, "utf-8"));

const parserPreset = await createPreset();
const parserOpts = parserPreset.parser;

const rules: Record<string, [number, string, unknown?]> = {};
for (const [name, value] of Object.entries(config.rules)) {
  if (Array.isArray(value)) {
    rules[name] = value as [number, string, unknown?];
  }
}

const results: LintResult[] = [];

for (const entry of corpus) {
  const ignored = isIgnored(entry.message, { defaults: true });

  if (ignored) {
    results.push({
      id: entry.id,
      valid: true,
      ignored: true,
      findings: [],
    });
    continue;
  }

  const outcome = await lint(entry.message, rules, {
    parserOpts,
  });

  const findings: FindingSummary[] = [];
  for (const e of outcome.errors) {
    findings.push({ rule: e.name, level: "error" });
  }
  for (const w of outcome.warnings) {
    findings.push({ rule: w.name, level: "warning" });
  }

  findings.sort((a, b) => {
    const cmp = a.rule.localeCompare(b.rule);
    return cmp !== 0 ? cmp : a.level.localeCompare(b.level);
  });

  results.push({
    id: entry.id,
    valid: outcome.valid,
    ignored: false,
    findings,
  });
}

console.log(JSON.stringify(results));
