import config from "@commitlint/config-conventional";
import createPreset from "conventional-changelog-conventionalcommits";

type EncodedValue =
  | { kind: "null" }
  | { kind: "bool"; bool: boolean }
  | { kind: "number"; number: string }
  | { kind: "string"; string: string }
  | { kind: "array"; items: EncodedValue[] }
  | { kind: "object"; object: Record<string, EncodedValue> }
  | { kind: "regexp"; source: string; flags: string }
  | { kind: "function" };

function encodeValue(value: unknown): EncodedValue {
  if (value === null) {
    return { kind: "null" };
  }

  if (value instanceof RegExp) {
    return { kind: "regexp", source: value.source, flags: value.flags };
  }

  if (Array.isArray(value)) {
    return { kind: "array", items: value.map((item) => encodeValue(item)) };
  }

  switch (typeof value) {
    case "boolean":
      return { kind: "bool", bool: value };
    case "number":
      return { kind: "number", number: String(value) };
    case "string":
      return { kind: "string", string: value };
    case "function":
      return { kind: "function" };
    case "object": {
      const object = Object.fromEntries(
        Object.entries(value).sort(([left], [right]) => left.localeCompare(right)).map(([key, entry]) => {
          return [key, encodeValue(entry)];
        }),
      );

      return { kind: "object", object };
    }
    default:
      throw new Error(`unsupported value type: ${typeof value}`);
  }
}

if (config.parserPreset !== "conventional-changelog-conventionalcommits") {
  throw new Error(`unexpected parser preset: ${String(config.parserPreset)}`);
}

const parserPreset = await createPreset();

const payload = {
  rules: Object.fromEntries(
    Object.entries(config.rules)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([ruleName, ruleValue]) => {
        return [ruleName, encodeValue(ruleValue)];
      }),
  ),
  parserPreset: {
    name: config.parserPreset,
    parserOpts: encodeValue(parserPreset.parser),
  },
};

console.log(JSON.stringify(payload));
