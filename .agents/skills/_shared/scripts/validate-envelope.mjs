#!/usr/bin/env node
// validate-envelope.mjs — subset JSON-Schema validator for the skillgrid
// pipeline envelopes (_shared/schemas/pipeline/), plus the in-band
// continuation rule from _shared/craft/envelope-contract.md.
//
// Usage: node validate-envelope.mjs <schema.json> <envelope.json | ->
// Exit: 0 valid · 1 invalid (findings JSON on stdout) · 2 usage.
//
// Supported keywords — the subset the pipeline schemas use. A schema that
// needs a keyword this script lacks is a change to this script + its test,
// not to the schema alone:
//   $ref (#/$defs/… only), type (incl. unions), const, enum, required,
//   properties, additionalProperties (bool | schema), propertyNames,
//   pattern, minLength, minimum, items (bool | schema), minItems, maxItems,
//   prefixItems, allOf, if/then/else.
//
// In-band rule (applied after schema validation, per envelope family):
//   gate-result:   allowed == false  → next.step must be runnable
//   gate-consent:  every choices[].next_step and off_path.step runnable
//   phase-status:  kind == "stop"    → continuation.step must be runnable
// runnable = non-empty; no <placeholder> segments; names a command
// (node|git|wt|bun|npm|npx|uv|uvx|gh|glab plus an argument) or a file with
// a known extension (optionally :line). A template is not runnable — that
// sub-rule is the load-bearing one (gentle-ai bench classifier, in_band).

import { readFileSync } from "node:fs";

function usage(msg) {
  console.error(`validate-envelope: ${msg}`);
  process.exit(2);
}

const [schemaArg, envelopeArg] = process.argv.slice(2);
if (!schemaArg || !envelopeArg) {
  usage("usage: validate-envelope.mjs <schema.json> <envelope.json | ->");
}

function readInput(name, arg) {
  try {
    return arg === "-" ? readFileSync(0, "utf8") : readFileSync(arg, "utf8");
  } catch (e) {
    usage(`cannot read ${name}: ${e.message}`);
  }
}

let schema, envelope;
try {
  schema = JSON.parse(readInput("schema", schemaArg));
} catch (e) {
  usage(`schema is not valid JSON: ${e.message}`);
}
try {
  envelope = JSON.parse(readInput("envelope", envelopeArg));
} catch (e) {
  usage(`envelope is not valid JSON: ${e.message}`);
}

const findings = [];

// --- subset validator ----------------------------------------------------------

function typeOf(value) {
  if (value === null) return "null";
  if (Array.isArray(value)) return "array";
  if (typeof value === "object") return "object";
  return typeof value;
}

function typeOk(type, value) {
  switch (type) {
    case "object": return value !== null && typeof value === "object" && !Array.isArray(value);
    case "array": return Array.isArray(value);
    case "string": return typeof value === "string";
    case "boolean": return typeof value === "boolean";
    case "integer": return Number.isInteger(value);
    case "number": return typeof value === "number";
    case "null": return value === null;
    default: return true;
  }
}

function validate(schemaNode, value, path, root) {
  if (schemaNode === true) return;
  if (schemaNode === false) {
    findings.push({ path, rule: "false", message: "no value allowed here" });
    return;
  }

  if (schemaNode.$ref) {
    const m = /^#\/\$defs\/([A-Za-z0-9_-]+)$/.exec(schemaNode.$ref);
    const target = m && root.$defs && root.$defs[m[1]];
    if (!target) {
      findings.push({ path, rule: "$ref", message: `unsupported or unresolved $ref ${schemaNode.$ref}` });
      return;
    }
    validate(target, value, path, root);
    return;
  }

  if (Array.isArray(schemaNode.allOf)) {
    for (const sub of schemaNode.allOf) validate(sub, value, path, root);
  }

  if (schemaNode.type !== undefined) {
    const types = Array.isArray(schemaNode.type) ? schemaNode.type : [schemaNode.type];
    if (!types.some((t) => typeOk(t, value))) {
      findings.push({ path, rule: "type", message: `expected ${types.join(" | ")}, got ${typeOf(value)}` });
      return; // deeper checks assume the declared type
    }
  }

  if (schemaNode.const !== undefined && JSON.stringify(value) !== JSON.stringify(schemaNode.const)) {
    findings.push({ path, rule: "const", message: `expected ${JSON.stringify(schemaNode.const)}, got ${JSON.stringify(value)}` });
  }

  if (Array.isArray(schemaNode.enum) && !schemaNode.enum.some((c) => JSON.stringify(c) === JSON.stringify(value))) {
    findings.push({ path, rule: "enum", message: `expected one of ${JSON.stringify(schemaNode.enum)}` });
  }

  if (typeof value === "string") {
    if (typeof schemaNode.minLength === "number" && value.length < schemaNode.minLength) {
      findings.push({ path, rule: "minLength", message: `string is shorter than ${schemaNode.minLength}` });
    }
    if (typeof schemaNode.pattern === "string" && !new RegExp(schemaNode.pattern).test(value)) {
      findings.push({ path, rule: "pattern", message: `does not match /${schemaNode.pattern}/` });
    }
  }

  if (typeof value === "number" && typeof schemaNode.minimum === "number" && value < schemaNode.minimum) {
    findings.push({ path, rule: "minimum", message: `below minimum ${schemaNode.minimum}` });
  }

  if (Array.isArray(value)) {
    if (typeof schemaNode.minItems === "number" && value.length < schemaNode.minItems) {
      findings.push({ path, rule: "minItems", message: `fewer than ${schemaNode.minItems} items` });
    }
    if (typeof schemaNode.maxItems === "number" && value.length > schemaNode.maxItems) {
      findings.push({ path, rule: "maxItems", message: `more than ${schemaNode.maxItems} items` });
    }
    const prefix = Array.isArray(schemaNode.prefixItems) ? schemaNode.prefixItems : [];
    value.forEach((item, i) => {
      if (i < prefix.length) {
        validate(prefix[i], item, `${path}[${i}]`, root);
      } else if (schemaNode.items === false) {
        findings.push({ path: `${path}[${i}]`, rule: "items", message: "no items beyond prefixItems" });
      } else if (schemaNode.items !== undefined) {
        validate(schemaNode.items, item, `${path}[${i}]`, root);
      }
    });
  }

  if (value !== null && typeof value === "object" && !Array.isArray(value)) {
    if (Array.isArray(schemaNode.required)) {
      for (const key of schemaNode.required) {
        if (!(key in value)) findings.push({ path: `${path}.${key}`, rule: "required", message: "missing required property" });
      }
    }
    if (schemaNode.propertyNames && typeof schemaNode.propertyNames.pattern === "string") {
      const re = new RegExp(schemaNode.propertyNames.pattern);
      for (const key of Object.keys(value)) {
        if (!re.test(key)) findings.push({ path: `${path}.${key}`, rule: "propertyNames", message: `key does not match /${schemaNode.propertyNames.pattern}/` });
      }
    }
    const props = schemaNode.properties || {};
    for (const [key, item] of Object.entries(value)) {
      if (key in props) {
        validate(props[key], item, `${path}.${key}`, root);
      } else if (schemaNode.additionalProperties === false) {
        findings.push({ path: `${path}.${key}`, rule: "additionalProperties", message: "unknown property" });
      } else if (schemaNode.additionalProperties && typeof schemaNode.additionalProperties === "object") {
        validate(schemaNode.additionalProperties, item, `${path}.${key}`, root);
      }
    }
  }

  if (schemaNode.if !== undefined) {
    const saved = findings.length;
    validate(schemaNode.if, value, path, root);
    const ifPassed = findings.length === saved;
    findings.length = saved; // the probe's findings are not violations
    const branch = ifPassed ? schemaNode.then : schemaNode.else;
    if (branch) validate(branch, value, path, root);
  }
}

// --- in-band continuation rule ---------------------------------------------------

const TEMPLATE = /<[A-Za-z_][A-Za-z0-9_]*>/;
const COMMAND = /(^|\s)(node|git|wt|bun|npm|npx|uv|uvx|gh|glab)\s+\S/;
const FILE = /\.(mjs|cjs|js|ts|py|md|yaml|yml|json|sh|txt|feature)(:\d+)?/;

function checkStep(path, step) {
  if (typeof step !== "string" || step.length === 0) {
    findings.push({ path, rule: "in-band", message: "denial carries no continuation step" });
    return;
  }
  if (TEMPLATE.test(step)) {
    findings.push({ path, rule: "in-band", message: `continuation is a template (${TEMPLATE.exec(step)[0]}): it is not runnable` });
  } else if (!COMMAND.test(step) && !FILE.test(step)) {
    findings.push({ path, rule: "in-band", message: "continuation names neither a command nor a file: not runnable" });
  }
}

function checkInBand(envelope) {
  if (!envelope || typeof envelope !== "object" || Array.isArray(envelope)) return;
  const family = envelope.schema;
  if (family === "skillgrid.gate-result/v1") {
    if (envelope.allowed === false) checkStep("$.next.step", envelope.next && envelope.next.step);
  } else if (family === "skillgrid.gate-consent/v1") {
    (Array.isArray(envelope.choices) ? envelope.choices : []).forEach((c, i) => {
      checkStep(`$.choices[${i}].next_step`, c && c.next_step);
    });
    checkStep("$.off_path.step", envelope.off_path && envelope.off_path.step);
  } else if (family === "skillgrid.phase-status/v1") {
    const nt = envelope.next_transition;
    if (nt && nt.kind === "stop") checkStep("$.next_transition.continuation.step", nt.continuation && nt.continuation.step);
  }
}

// --- run --------------------------------------------------------------------------

validate(schema, envelope, "$", schema);
checkInBand(envelope);

const out = { valid: findings.length === 0, findings };
console.log(JSON.stringify(out, null, 2));
process.exit(findings.length ? 1 : 0);
