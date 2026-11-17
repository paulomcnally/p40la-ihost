/**
 * Parses YAML configuration files.
 *
 * Supports YAML 1.2 block and flow collections, indentless sequence values,
 * multiline plain and quoted scalars, block scalars, anchors, aliases, and
 * YAML 1.1 merge keys (`<<`).
 *
 * Unlike YAML 1.2, the closing bracket of a multiline flow collection may sit
 * at its parent's indentation.
 *
 * Unsupported: compact nested block sequences (`- - value` on one line),
 * multiline quoted scalars in flow collections, explicit complex keys,
 * tag/directive processing, recursive aliases, and document streams. Mapping
 * keys are JavaScript strings.
 *
 * @stability unstable
 * @since 4.0.0
 */
/**
 * Parses one YAML document into JavaScript values.
 *
 * **Details**
 *
 * The core YAML 1.2 scalar schema is used, so booleans, nulls, and numbers are
 * decoded while date-like values remain strings.
 *
 * YAML 1.1 merge keys are always enabled. An unquoted `<<` key merges a
 * mapping, or a sequence of mappings, into the enclosing mapping. Explicit
 * keys take precedence over merged keys, earlier merge sources over later
 * ones, and a mapping may contain only one merge key. Quote the key (`"<<"`)
 * to keep it as an ordinary key.
 *
 * @stability unstable
 * @category decoding
 * @since 4.0.0
 */
export declare const parse: (input: string) => unknown;
//# sourceMappingURL=Yaml.d.ts.map