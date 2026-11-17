/**
 * Formats and presents CLI help, errors, and version information.
 *
 * The `Formatter` service turns help documents, errors, and version
 * information into strings. The `Presenter` service decides where the built-in
 * help, version, and invalid-invocation output of `Command.runWith` is written,
 * and can be replaced to suppress or redirect it.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Console from "../Console.js";
import * as Context from "../Context.js";
import * as Effect from "../Effect.js";
import * as Layer from "../Layer.js";
import * as Option from "../Option.js";
import * as Ansi from "./internal/ansi.js";
import * as TerminalCapabilities from "./TerminalCapabilities.js";
/**
 * Service reference for the CLI output formatter. Provides a default implementation
 * that can be overridden for custom formatting or testing.
 *
 * **Example** (Accessing the output formatter)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { CliOutput } from "effect/cli"
 *
 * // Access the formatter service
 * const program = Effect.gen(function*() {
 *   const formatter = yield* CliOutput.Formatter
 *
 *   // Format version information
 *   return formatter.formatVersion("my-cli", "2.1.0")
 * })
 *
 * // Run with default formatter
 * await Effect.runPromise(program.pipe(
 *   Effect.provide(CliOutput.layer(CliOutput.defaultFormatter({ colors: false })))
 * )) // => "my-cli v2.1.0"
 * ```
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export const Formatter = /*#__PURE__*/Context.Reference("effect/cli/CliOutput", {
  defaultValue: () => defaultFormatter()
});
/**
 * Default presenter that renders events with the current `Formatter` and
 * writes them with `Console`.
 *
 * **Details**
 *
 * Help and version output goes to `Console.log`. Invalid invocations write
 * help to `Console.log` and errors to `Console.error`. When no `Formatter` is
 * installed, the default formatter colors output when `TerminalCapabilities`
 * says it can.
 *
 * @see {@link Presenter} for replacing the default output
 * @stability unstable
 * @category defaults
 * @since 4.0.0
 */
export const defaultPresenter = {
  present: /*#__PURE__*/Effect.fnUntraced(function* (event) {
    const context = yield* Effect.context();
    const formatter = Context.getOrUndefined(context, Formatter) ?? defaultFormatter({
      colors: Context.get(context, TerminalCapabilities.TerminalCapabilities).canColor
    });
    switch (event._tag) {
      case "Help":
        return yield* Console.log(formatter.formatHelpDoc(event.helpDoc));
      case "InvalidInvocation":
        yield* Console.log(formatter.formatHelpDoc(event.helpDoc));
        return yield* Console.error(formatter.formatErrors(event.errors));
      case "Version":
        return yield* Console.log(formatter.formatVersion(event.name, event.version));
    }
  })
};
/**
 * Service reference for the CLI output presenter. Defaults to
 * `defaultPresenter`.
 *
 * @see {@link layerPresenter} for providing a custom presenter
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export const Presenter = /*#__PURE__*/Context.Reference("effect/cli/CliOutput/Presenter", {
  defaultValue: () => defaultPresenter
});
/**
 * Creates a Layer that provides a custom Formatter implementation.
 *
 * **Example** (Providing a custom formatter)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { CliOutput } from "effect/cli"
 *
 * // Create a custom formatter without colors
 * const noColorFormatter = CliOutput.defaultFormatter({ colors: false })
 * const NoColorLayer = CliOutput.layer(noColorFormatter)
 *
 * // Create a program that uses the custom formatter
 * const program = Effect.gen(function*() {
 *   const formatter = yield* CliOutput.Formatter
 *   return formatter.formatVersion("my-cli", "1.0.0")
 * }).pipe(
 *   Effect.provide(NoColorLayer)
 * )
 *
 * await Effect.runPromise(program) // => "my-cli v1.0.0"
 * ```
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layer = formatter => Layer.succeed(Formatter)(formatter);
/**
 * Creates a Layer that provides a custom Presenter implementation.
 *
 * @see {@link Presenter} for the events a presenter receives
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layerPresenter = presenter => Layer.succeed(Presenter)(presenter);
const escapeControlCharacters = text =>
// oxlint-disable-next-line no-control-regex
text.replace(/[\u0000-\u0008\u000B\u000C\u000E-\u001F\u007F]/g, character => `\\x${character.charCodeAt(0).toString(16).padStart(2, "0")}`);
/**
 * Creates a default formatter with configurable options.
 *
 * **Details**
 *
 * - `colors`: force ANSI colors on or off. Defaults to terminal detection.
 * - `inheritedFlags`: how subcommand help renders flags inherited from
 *   ancestor commands. `"full"` (default) lists them under `INHERITED FLAGS`,
 *   `"summary"` replaces the section with a pointer to the parent command's
 *   help, and `"none"` omits it.
 * - `globalFlags`: the same choice for `GLOBAL FLAGS`, with the summary
 *   pointing at the help of the nearest command that declares global flags
 *   (the root command for built-in flags). Commands that declare global flags
 *   always list them in full.
 *
 * **Gotchas**
 *
 * Per-flag filtering in a custom formatter matches names, not flag identities.
 * Renaming a flag without updating the filter silently stops it from matching.
 *
 * **Example** (Creating default formatters)
 *
 * ```ts import.meta.vitest
 * import { CliError, CliOutput } from "effect/cli"
 *
 * // Create a formatter without colors for tests or CI environments
 * const noColorFormatter = CliOutput.defaultFormatter({ colors: false })
 *
 * // Create a formatter with colors forced on
 * const colorFormatter = CliOutput.defaultFormatter({ colors: true })
 *
 * // Auto-detect colors based on terminal support (default behavior)
 * const autoFormatter = CliOutput.defaultFormatter()
 *
 * // Point subcommand help at the declaring scope for global flags
 * const compactFormatter = CliOutput.defaultFormatter({ globalFlags: "summary" })
 *
 * const error = new CliError.InvalidValue({
 *   option: "foo",
 *   value: "bar",
 *   expected: "baz",
 *   kind: "flag"
 * })
 *
 * noColorFormatter.formatError(error).includes("Invalid value") // => true
 * colorFormatter.formatVersion("my-tool", "1.2.3").includes("my-tool") // => true
 * autoFormatter.formatVersion("my-tool", "1.2.3").includes("1.2.3") // => true
 * compactFormatter.formatVersion("my-tool", "1.2.3").includes("my-tool") // => true
 * ```
 *
 * **Example** (Hiding one inherited flag)
 *
 * Filter only `inheritedFlags` to keep `--profile` visible on its declaring
 * command while hiding it on descendants. Other inherited flags still render.
 *
 * ```ts import.meta.vitest
 * import { Context, Effect, Option } from "effect"
 * import { CliOutput } from "effect/cli"
 *
 * const base = CliOutput.defaultFormatter({ colors: false })
 * const formatter: CliOutput.Formatter = {
 *   ...base,
 *   formatHelpDoc: (doc) =>
 *     base.formatHelpDoc({
 *       ...doc,
 *       inheritedFlags: doc.inheritedFlags?.filter((flag) => flag.name !== "profile")
 *     })
 * }
 *
 * const program = Effect.gen(function*() {
 *   const output = yield* CliOutput.Formatter
 *   return output.formatHelpDoc({
 *     description: "Deploy the application",
 *     usage: "app deploy",
 *     annotations: Context.empty(),
 *     flags: [],
 *     inheritedFlags: ["profile", "region"].map((name) => ({
 *       name,
 *       aliases: [],
 *       type: "string",
 *       description: Option.none(),
 *       required: false
 *     }))
 *   })
 * }).pipe(Effect.provide(CliOutput.layer(formatter)))
 *
 * const help = await Effect.runPromise(program)
 * help.includes("--profile") // => false
 * help.includes("--region") // => true
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const defaultFormatter = options => {
  const useColor = options?.colors ?? Ansi.detectColors();
  const helpOptions = {
    inheritedFlags: options?.inheritedFlags ?? "full",
    globalFlags: options?.globalFlags ?? "full"
  };
  // Color palette using ANSI escape codes
  const colors = useColor ? {
    bold: text => `\x1b[1m${text}\x1b[0m`,
    dim: text => `\x1b[2m${text}\x1b[0m`,
    cyan: text => `\x1b[36m${text}\x1b[0m`,
    green: text => `\x1b[32m${text}\x1b[0m`,
    blue: text => `\x1b[34m${text}\x1b[0m`,
    yellow: text => `\x1b[33m${text}\x1b[0m`,
    magenta: text => `\x1b[35m${text}\x1b[0m`
  } : {
    bold: text => text,
    dim: text => text,
    cyan: text => text,
    green: text => text,
    blue: text => text,
    yellow: text => text,
    magenta: text => text
  };
  const reset = useColor ? "\x1b[0m" : "";
  const red = useColor ? "\x1b[31m" : "";
  const bold = useColor ? "\x1b[1m" : "";
  return {
    formatHelpDoc: doc => formatHelpDocImpl(doc, colors, helpOptions),
    formatCliError: error => escapeControlCharacters(error.message),
    formatError: error => {
      return `\n${bold}${red}ERROR${reset}\n  ${escapeControlCharacters(error.message)}${reset}`;
    },
    formatErrors: errors => {
      if (errors.length === 0) return "";
      if (errors.length === 1) {
        return `\n${bold}${red}ERROR${reset}\n  ${escapeControlCharacters(errors[0].message)}${reset}`;
      }
      // Group errors by _tag
      const grouped = new Map();
      for (const error of errors) {
        const tag = error._tag ?? "Error";
        const group = grouped.get(tag) ?? [];
        group.push(error);
        grouped.set(tag, group);
      }
      const sections = [];
      sections.push(`\n${bold}${red}ERRORS${reset}`);
      for (const [, group] of grouped) {
        for (const error of group) {
          sections.push(`  ${escapeControlCharacters(error.message)}${reset}`);
        }
      }
      return sections.join("\n");
    },
    formatVersion: (name, version) => `${colors.bold(name)} ${colors.dim("v")}${colors.bold(version)}`
  };
};
/**
 * Strips ANSI escape codes from a string to calculate visual width.
 * @internal
 */
const stripAnsi = text => {
  // oxlint-disable-next-line no-control-regex
  return text.replace(/\u001B\[[0-9;]*m/g, "");
};
/**
 * Segments text into grapheme clusters.
 * @internal
 */
const graphemeSegmenter = /*#__PURE__*/new Intl.Segmenter(undefined, {
  granularity: "grapheme"
});
const zeroWidthCodePoints = /[\p{Mark}\p{Control}\p{Default_Ignorable_Code_Point}]/gu;
const emojiPresentation = /\p{Emoji_Presentation}/u;
const isFullWidthCodePoint = codePoint => {
  if (codePoint < 0x1100) return false;
  return codePoint <= 0x115f || codePoint === 0x2329 || codePoint === 0x232a || codePoint >= 0x2e80 && codePoint <= 0x303e || codePoint >= 0x3040 && codePoint <= 0xa4cf || codePoint >= 0xac00 && codePoint <= 0xd7a3 || codePoint >= 0xf900 && codePoint <= 0xfaff || codePoint >= 0xfe10 && codePoint <= 0xfe19 || codePoint >= 0xfe30 && codePoint <= 0xfe6f || codePoint >= 0xff00 && codePoint <= 0xff60 || codePoint >= 0xffe0 && codePoint <= 0xffe6 || codePoint >= 0x1b000 && codePoint <= 0x1b2ff || codePoint >= 0x1f200 && codePoint <= 0x1f251 || codePoint >= 0x20000 && codePoint <= 0x3fffd;
};
const graphemeWidth = grapheme => {
  const visible = grapheme.replace(zeroWidthCodePoints, "");
  if (visible.length === 0) return 0;
  if (emojiPresentation.test(grapheme) || grapheme.includes("\uFE0F")) return 2;
  return isFullWidthCodePoint(visible.codePointAt(0)) ? 2 : 1;
};
/**
 * Measures the number of terminal cells a string occupies.
 *
 * **Details**
 *
 * Uses the same measurement as built-in help and table rendering. ANSI styling
 * is stripped before measuring graphemes: zero-width graphemes count as `0`,
 * emoji presentation sequences and fullwidth characters as `2`, and other
 * graphemes as `1`.
 *
 * **Example** (Measuring terminal-cell widths)
 *
 * ```ts import.meta.vitest
 * import { CliOutput } from "effect/cli"
 *
 * CliOutput.displayWidth("file") // => 4
 * CliOutput.displayWidth("ファイル") // => 8
 * CliOutput.displayWidth("é") // => 1
 * CliOutput.displayWidth("1️⃣") // => 2
 * CliOutput.displayWidth("\u001B[1mbold\u001B[0m") // => 4
 * ```
 *
 * @stability unstable
 * @category measuring
 * @since 4.0.3
 */
export const displayWidth = text => {
  let length = 0;
  for (const {
    segment
  } of graphemeSegmenter.segment(stripAnsi(text))) {
    length += graphemeWidth(segment);
  }
  return length;
};
/**
 * Helper function to pad strings to a specified width.
 * @internal
 */
const pad = (s, width) => {
  const actualLength = displayWidth(s);
  const padding = Math.max(0, width - actualLength);
  return s + " ".repeat(padding);
};
/**
 * Renders a table with aligned columns.
 * @internal
 */
const renderTable = (rows, widthCap) => {
  const maxColumn = Math.max(...rows.map(r => displayWidth(r.left))) + 4;
  const col = widthCap === undefined ? maxColumn : Math.min(maxColumn, widthCap);
  return rows.map(({
    left,
    right
  }) => `  ${pad(left, Math.max(col, displayWidth(left) + 1))}${right}`).join("\n");
};
const formatSubcommandName = (name, alias) => alias ? `${name}, ${alias}` : name;
/**
 * Internal implementation of help formatting that accepts configurable color functions.
 * @internal
 */
const formatHelpDocImpl = (doc, colors, options) => {
  const sections = [];
  // Description section
  if (doc.description) {
    sections.push(colors.bold("DESCRIPTION"));
    sections.push(`  ${doc.description}`);
    sections.push("");
  }
  // Usage section
  sections.push(colors.bold("USAGE"));
  sections.push(`  ${colors.cyan(doc.usage)}`);
  sections.push("");
  // Arguments section
  if (doc.args && doc.args.length > 0) {
    sections.push(colors.bold("ARGUMENTS"));
    const argRows = doc.args.map(arg => {
      let name = arg.name;
      if (arg.variadic) {
        name += "...";
      }
      const coloredName = colors.green(name);
      const coloredType = colors.dim(arg.type);
      const nameType = `${coloredName} ${coloredType}`;
      const optionalSuffix = arg.required ? "" : colors.dim(" (optional)");
      const description = Option.getOrElse(arg.description, () => "") + optionalSuffix;
      return {
        left: nameType,
        right: description
      };
    });
    sections.push(renderTable(argRows, 25));
    sections.push("");
  }
  const renderFlags = (title, flags) => {
    sections.push(colors.bold(title));
    sections.push(renderTable(flags.map(flag => {
      const names = [`--${flag.name}`, ...flag.aliases].map(colors.green).join(", ");
      const typePart = flag.type !== "boolean" ? ` ${colors.dim(flag.type)}` : "";
      return {
        left: names + typePart,
        right: Option.getOrElse(flag.description, () => "")
      };
    })));
    sections.push("");
  };
  const summaries = [];
  const renderFlagSection = (title, flags, mode, summaryPath) => {
    if (!flags || flags.length === 0 || mode === "none") return;
    if (mode === "summary" && summaryPath && summaryPath.length > 0) {
      summaries.push(`Run '${summaryPath.join(" ")} --help' for ${title.toLowerCase()}`);
    } else {
      renderFlags(title, flags);
    }
  };
  if (doc.flags.length > 0) {
    renderFlags("FLAGS", doc.flags);
  }
  const commandPath = doc.commandPath;
  renderFlagSection("INHERITED FLAGS", doc.inheritedFlags, options.inheritedFlags, commandPath?.slice(0, -1));
  // Keep globals discoverable on their declaring command, even in "none" mode.
  const globalFlagsScope = doc.globalFlagsScope ?? commandPath?.slice(0, 1);
  const declaresGlobalFlags = commandPath === undefined || globalFlagsScope?.length === commandPath.length;
  renderFlagSection("GLOBAL FLAGS", doc.globalFlags, declaresGlobalFlags ? "full" : options.globalFlags, globalFlagsScope);
  if (summaries.length > 0) {
    sections.push(...summaries, "");
  }
  // Subcommands section
  if (doc.subcommands && doc.subcommands.length > 0) {
    const ungrouped = doc.subcommands.find(group => group.group === undefined);
    if (ungrouped) {
      sections.push(colors.bold("SUBCOMMANDS"));
      sections.push(renderTable(ungrouped.commands.map(sub => ({
        left: colors.cyan(formatSubcommandName(sub.name, sub.alias)),
        right: sub.shortDescription ?? sub.description
      })), 20));
      if (doc.subcommands.length > 1) {
        sections.push("");
      }
    }
    for (const group of doc.subcommands) {
      if (group.group === undefined) continue;
      sections.push(colors.bold(`${group.group}:`));
      sections.push(renderTable(group.commands.map(sub => ({
        left: colors.cyan(formatSubcommandName(sub.name, sub.alias)),
        right: sub.shortDescription ?? sub.description
      })), 20));
      sections.push("");
    }
  }
  // Examples section
  if (doc.examples && doc.examples.length > 0) {
    sections.push(colors.bold("EXAMPLES"));
    let first = true;
    let previousHadDescription = false;
    for (const example of doc.examples) {
      if (example.description) {
        if (!first) sections.push("");
        sections.push(`  ${colors.dim(`# ${example.description}`)}`);
      } else if (previousHadDescription) {
        sections.push("");
      }
      sections.push(`  ${colors.cyan(example.command)}`);
      first = false;
      previousHadDescription = !!example.description;
    }
    sections.push("");
  }
  // Remove trailing empty line if present
  if (sections[sections.length - 1] === "") {
    sections.pop();
  }
  return sections.join("\n");
};
//# sourceMappingURL=CliOutput.js.map