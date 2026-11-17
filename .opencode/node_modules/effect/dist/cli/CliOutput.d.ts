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
import * as Context from "../Context.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
import type * as CliError from "./CliError.ts";
import type { HelpDoc } from "./HelpDoc.ts";
/**
 * Defines the service interface for formatting CLI output including help, errors, and version info.
 * This allows customization of output formatting, including color support.
 *
 * **Example** (Customizing CLI output formatting)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { CliOutput } from "effect/cli"
 *
 * // Create a custom formatter implementation
 * const customFormatter: CliOutput.Formatter = {
 *   formatHelpDoc: (doc) => `Custom Help: ${doc.usage}`,
 *   formatCliError: (error) => `Error: ${error.message}`,
 *   formatError: (error) => `[ERROR] ${error.message}`,
 *   formatVersion: (name, version) => `${name} (${version})`,
 *   formatErrors: (errors) => errors.map((error) => error.message).join("\\n")
 * }
 *
 * // Use the custom formatter in a program
 * const program = Effect.gen(function*() {
 *   const formatter = yield* CliOutput.Formatter
 *   return formatter.formatVersion("myapp", "1.0.0")
 * }).pipe(
 *   Effect.provide(CliOutput.layer(customFormatter))
 * )
 *
 * await Effect.runPromise(program) // => "myapp (1.0.0)"
 * ```
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface Formatter {
    /**
     * Formats a HelpDoc structure into a readable string format.
     *
     * **Example** (Formatting help documents)
     *
     * ```ts import.meta.vitest
     * import { Option as O } from "effect"
     * import { CliOutput } from "effect/cli"
     * import type { HelpDoc } from "effect/cli"
     *
     * const helpDoc: HelpDoc = {
     *   usage: "myapp [options] <file>",
     *   description: "Process files with various options",
     *   flags: [
     *     {
     *       name: "verbose",
     *       aliases: ["-v"],
     *       type: "boolean",
     *       description: O.some("Enable verbose output"),
     *       required: false
     *     }
     *   ],
     *   args: [
     *     {
     *       name: "file",
     *       type: "string",
     *       description: O.some("Input file to process"),
     *       required: true,
     *       variadic: false
     *     }
     *   ]
     * }
     *
     * const formatter = CliOutput.defaultFormatter()
     * const helpText = formatter.formatHelpDoc(helpDoc)
     * const sectionsPresent = [helpText.includes("DESCRIPTION"), helpText.includes("FLAGS")] // => [true, true]
     * ```
     *
     * @stability unstable
     * @since 4.0.0
     */
    readonly formatHelpDoc: (doc: HelpDoc) => string;
    /**
     * Formats a CLI error for display. Default implementation mirrors the error message.
     *
     * **Example** (Formatting CLI errors)
     *
     * ```ts import.meta.vitest
     * import { Data } from "effect"
     * import { CliOutput } from "effect/cli"
     *
     * class InvalidOption extends Data.TaggedError("InvalidOption")<{
     *   readonly message: string
     * }> {}
     *
     * const formatter = CliOutput.defaultFormatter()
     * const error = new InvalidOption({ message: "Unknown flag '--invalid'" })
     * const errorMessage = formatter.formatCliError(error)
     * errorMessage // => "Unknown flag '--invalid'"
     * ```
     *
     * @stability unstable
     * @since 4.0.0
     */
    readonly formatCliError: (error: CliError.CliError) => string;
    /**
     * Formats an error section with proper styling and color reset.
     *
     * **Example** (Formatting error sections)
     *
     * ```ts import.meta.vitest
     * import { Data } from "effect"
     * import { CliOutput } from "effect/cli"
     *
     * class ValidationError extends Data.TaggedError("ValidationError")<{
     *   readonly message: string
     * }> {}
     *
     * const colorFormatter = CliOutput.defaultFormatter({ colors: true })
     * const noColorFormatter = CliOutput.defaultFormatter({ colors: false })
     *
     * const error = new ValidationError({ message: "Value must be positive" })
     *
     * const coloredError = colorFormatter.formatError(error)
     * coloredError.includes("\u001b[31mERROR") // => true
     *
     * const plainError = noColorFormatter.formatError(error)
     * plainError // => "\nERROR\n  Value must be positive"
     * ```
     *
     * @stability unstable
     * @since 4.0.0
     */
    readonly formatError: (error: CliError.CliError) => string;
    /**
     * Formats version output for display.
     *
     * **Example** (Formatting version output)
     *
     * ```ts import.meta.vitest
     * import { CliOutput } from "effect/cli"
     *
     * const colorFormatter = CliOutput.defaultFormatter({ colors: true })
     * const noColorFormatter = CliOutput.defaultFormatter({ colors: false })
     *
     * const appName = "my-awesome-tool"
     * const version = "1.2.3"
     *
     * const coloredVersion = colorFormatter.formatVersion(appName, version)
     * coloredVersion // => "\u001b[1mmy-awesome-tool\u001b[0m \u001b[2mv\u001b[0m\u001b[1m1.2.3\u001b[0m"
     *
     * const plainVersion = noColorFormatter.formatVersion(appName, version)
     * plainVersion // => "my-awesome-tool v1.2.3"
     * ```
     *
     * @stability unstable
     * @since 4.0.0
     */
    readonly formatVersion: (name: string, version: string) => string;
    /**
     * Formats multiple CLI errors for display, grouping by error type.
     *
     * **Example** (Formatting grouped errors)
     *
     * ```ts import.meta.vitest
     * import { CliError, CliOutput } from "effect/cli"
     *
     * const formatter = CliOutput.defaultFormatter({ colors: false })
     *
     * const errors = [
     *   new CliError.UnrecognizedOption({
     *     option: "--foo",
     *     suggestions: ["--force"]
     *   }),
     *   new CliError.UnrecognizedOption({ option: "--bar", suggestions: [] }),
     *   new CliError.MissingOption({ option: "--required" })
     * ]
     *
     * const output = formatter.formatErrors(errors)
     * const optionsPresent = [output.includes("--foo"), output.includes("--required")] // => [true, true]
     * ```
     *
     * @stability unstable
     * @since 4.0.0
     */
    readonly formatErrors: (errors: ReadonlyArray<CliError.CliError>) => string;
}
/**
 * Presents the built-in help, version, and invalid-invocation output of
 * `Command.runWith`.
 *
 * **When to use**
 *
 * Use to suppress, redirect, or replace that output, for example to keep
 * stdout clean for machine-readable results.
 *
 * **Details**
 *
 * Events carry the help document and errors rather than rendered text, so a
 * presenter can still render them with the `Formatter` service. Presenting an
 * event does not change the result of the run: implicit help and invalid
 * invocations still fail with `CliError.ShowHelp`.
 *
 * **Gotchas**
 *
 * `--completions` scripts, wizard mode output, and handler failures rendered
 * through `CliError.userMessage` do not go through the presenter.
 *
 * **Example** (Writing invalid-invocation output to stderr)
 *
 * ```ts import.meta.vitest
 * import { Console, Effect } from "effect"
 * import { CliOutput } from "effect/cli"
 *
 * const StderrPresenter = CliOutput.layerPresenter({
 *   present: (event) =>
 *     event._tag === "InvalidInvocation"
 *       ? Effect.gen(function*() {
 *         const formatter = yield* CliOutput.Formatter
 *         yield* Console.error(formatter.formatHelpDoc(event.helpDoc))
 *         yield* Console.error(formatter.formatErrors(event.errors))
 *       })
 *       : CliOutput.defaultPresenter.present(event)
 * })
 * ```
 *
 * @see {@link defaultPresenter} for the default console output
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface Presenter {
    /**
     * Presents a CLI output event.
     *
     * @stability unstable
     * @since 4.0.0
     */
    readonly present: (event: Presenter.Event) => Effect.Effect<void>;
}
/**
 * Types used by the `Presenter` service.
 *
 * @stability unstable
 * @since 4.0.0
 */
export declare namespace Presenter {
    /**
     * Output events produced while running a CLI command.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    type Event = Help | InvalidInvocation | Version;
    /**
     * Help for a command path.
     *
     * **Details**
     *
     * `reason` is `"Requested"` for `--help`, and `"Implicit"` when the run
     * fails with a `CliError.ShowHelp` that carries no errors, such as selecting
     * a command that has no handler, or when `renderErrors` is `false`.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    interface Help {
        readonly _tag: "Help";
        readonly reason: "Requested" | "Implicit";
        readonly commandPath: ReadonlyArray<string>;
        readonly helpDoc: HelpDoc;
    }
    /**
     * Help and parse or validation errors for an invalid invocation.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    interface InvalidInvocation {
        readonly _tag: "InvalidInvocation";
        readonly commandPath: ReadonlyArray<string>;
        readonly helpDoc: HelpDoc;
        readonly errors: ReadonlyArray<CliError.NonShowHelpErrors>;
    }
    /**
     * Version information requested with `--version`.
     *
     * @stability unstable
     * @category models
     * @since 4.0.0
     */
    interface Version {
        readonly _tag: "Version";
        readonly name: string;
        readonly version: string;
    }
}
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
export declare const Formatter: Context.Reference<Formatter>;
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
export declare const defaultPresenter: Presenter;
/**
 * Service reference for the CLI output presenter. Defaults to
 * `defaultPresenter`.
 *
 * @see {@link layerPresenter} for providing a custom presenter
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare const Presenter: Context.Reference<Presenter>;
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
export declare const layer: (formatter: Formatter) => Layer.Layer<never>;
/**
 * Creates a Layer that provides a custom Presenter implementation.
 *
 * @see {@link Presenter} for the events a presenter receives
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerPresenter: (presenter: Presenter) => Layer.Layer<never>;
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
export declare const defaultFormatter: (options?: {
    readonly colors?: boolean | undefined;
    readonly inheritedFlags?: "full" | "summary" | "none" | undefined;
    readonly globalFlags?: "full" | "summary" | "none" | undefined;
}) => Formatter;
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
export declare const displayWidth: (text: string) => number;
//# sourceMappingURL=CliOutput.d.ts.map