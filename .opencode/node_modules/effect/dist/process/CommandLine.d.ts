import * as Redacted from "../Redacted.ts";
import * as Result from "../Result.ts";
/**
 * Command line syntax understood by the functions in this module.
 *
 * **Details**
 *
 * - `"posix"`: POSIX shells (`sh`, `dash`, `bash`, `zsh`). Quoted words are
 *   also safe in `fish`.
 * - `"powershell"`: PowerShell 7.3 or later, which passes arguments to native
 *   programs unchanged (`$PSNativeCommandArgumentPassing` set to `Standard` or
 *   `Windows`).
 * - `"windows"`: the Microsoft C runtime convention, also implemented by
 *   `CommandLineToArgvW`, that most Windows programs use. It is not the syntax
 *   of `cmd.exe` or of batch files.
 *
 * @stability unstable
 * @category models
 * @since 4.0.7
 */
export type Format = "posix" | "powershell" | "windows";
/**
 * One argument: a plain string or a `Redacted` string that must not appear in
 * logs.
 *
 * @stability unstable
 * @category models
 * @since 4.0.7
 */
export type Word = string | Redacted.Redacted<string>;
/**
 * Single value accepted inside a `${}` interpolation of {@link template}.
 *
 * @stability unstable
 * @category models
 * @since 4.0.7
 */
export type TemplateExpressionItem = string | number | boolean | Redacted.Redacted<string>;
/**
 * Value accepted inside a `${}` interpolation of {@link template}.
 *
 * **Details**
 *
 * An array adds one argument per element and must stand alone as a whole
 * argument. An empty array adds no arguments.
 *
 * @stability unstable
 * @category models
 * @since 4.0.7
 */
export type TemplateExpression = TemplateExpressionItem | ReadonlyArray<TemplateExpressionItem>;
declare const CommandLineError_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "CommandLineError";
} & Readonly<A>;
/**
 * Error returned when a command line or a template cannot be built.
 *
 * **Details**
 *
 * - `NulCharacter`: an argument passed to {@link quote} or {@link join}
 *   contains a NUL character, which no command line can carry.
 * - `InvalidArgument`: the format cannot carry an argument: a Windows program
 *   name with a double quote, or the argument `--%`, which PowerShell drops
 *   when it calls a native program.
 * - `InvalidEscape`: template text ends with a backslash that escapes nothing,
 *   or contains an invalid JavaScript escape sequence.
 * - `UnterminatedQuote`: a quoted section of a template is not closed.
 * - `UnsupportedSyntax`: template text contains shell syntax that cannot work
 *   without a shell, such as operators, expansions, globs, comments or
 *   variable assignments, or an interpolated array is not a separate argument.
 *
 * Messages never include the input, which may contain secrets.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.7
 */
export declare class CommandLineError extends CommandLineError_base<{
    readonly reason: "NulCharacter" | "InvalidArgument" | "InvalidEscape" | "UnterminatedQuote" | "UnsupportedSyntax";
    readonly message: string;
}> {
}
/**
 * Quotes one argument so that a program reading the given format sees it as
 * exactly one argument, with one PowerShell exception listed under Gotchas.
 *
 * **When to use**
 *
 * Use when you insert an argument into a command line that is otherwise
 * written by hand, for example a script passed to `sh -c`.
 *
 * **Details**
 *
 * - `"posix"` leaves words made of letters, digits and `_ @ + : , . / -`
 *   unchanged and wraps everything else in single quotes. Single quotes and
 *   backslashes are written outside the quotes, so `fish` reads the result
 *   the same way.
 * - `"powershell"` always uses single quotes, doubling `'` and the
 *   typographic single quotes that PowerShell also accepts as quotes.
 * - `"windows"` quotes words that contain a space, a tab, a line break or a
 *   double quote, using the C runtime backslash rules. Use {@link join} for a
 *   complete command line, because the program name follows different rules.
 *
 * **Gotchas**
 *
 * - The result contains the value of a `Redacted` argument. Use
 *   {@link display} for logs.
 * - Control characters are kept as they are. That is safe for `sh -c` and SSH
 *   commands, but not for text typed into an interactive shell.
 * - PowerShell never passes an argument equal to `--%` to a native program,
 *   however it is quoted. Use {@link join} for a whole native call, which
 *   fails with `InvalidArgument` instead.
 *
 * **Example** (Quoting arguments for different shells)
 *
 * ```ts import.meta.vitest
 * import { Result } from "effect"
 * import { CommandLine } from "effect/process"
 *
 * CommandLine.quote("it's") // => Result.succeed("'it'\\''s'")
 * CommandLine.quote("it's", { format: "powershell" }) // => Result.succeed("'it''s'")
 * CommandLine.quote("C:\\My Files\\", { format: "windows" }) // => Result.succeed("\"C:\\My Files\\\\\"")
 * ```
 *
 * @see {@link join} for quoting a whole argument list
 *
 * @stability unstable
 * @category encoding
 * @since 4.0.7
 */
export declare const quote: (word: Word, options?: {
    readonly format?: Format | undefined;
}) => Result.Result<string, CommandLineError>;
/**
 * Builds a command line that runs the given program and arguments.
 *
 * **When to use**
 *
 * Use when a transport accepts only one string, such as an SSH exec request or
 * a `shell` option, and the arguments must reach the program unchanged.
 *
 * **Details**
 *
 * The first word is the program. Each word is quoted as in {@link quote} and
 * the words are separated by spaces.
 *
 * - `"posix"` also quotes a program name that `sh`, `bash`, `zsh` or `fish`
 *   reads as syntax, such as `if` or `time`.
 * - `"powershell"` starts the line with the call operator `&`, because a
 *   quoted program name is otherwise read as a string value.
 * - `"windows"` quotes the program name with the C runtime rules for program
 *   names, which do not support backslash escapes.
 * - An empty list returns an empty string.
 *
 * **Gotchas**
 *
 * - The result contains the values of `Redacted` arguments. Use
 *   {@link display} for logs.
 * - A Windows program name that contains `"` fails with `InvalidArgument`.
 * - For `"powershell"`, an argument equal to `--%` fails with
 *   `InvalidArgument`: PowerShell drops it when it calls a native program,
 *   however it is quoted.
 * - `"windows"` output is the command line of a program, not a `cmd.exe`
 *   script. `cmd.exe`, which `shell: true` uses on Windows, expands `%NAME%`
 *   and reads `^ & | < >` before the program sees its arguments, so a script
 *   that contains untrusted values has to be written by hand.
 *
 * **Example** (Joining arguments into command lines)
 *
 * ```ts import.meta.vitest
 * import { Result } from "effect"
 * import { CommandLine } from "effect/process"
 *
 * const args = ["git", "commit", "-m", "fix: don't fail"]
 *
 * CommandLine.join(args) // => Result.succeed("git commit -m 'fix: don'\\''t fail'")
 * CommandLine.join(args, { format: "powershell" }) // => Result.succeed("& 'git' 'commit' '-m' 'fix: don''t fail'")
 * CommandLine.join(args, { format: "windows" }) // => Result.succeed("git commit -m \"fix: don't fail\"")
 * ```
 *
 * @see {@link display} for rendering a command for logs and error messages
 *
 * @stability unstable
 * @category encoding
 * @since 4.0.7
 */
export declare const join: (words: ReadonlyArray<Word>, options?: {
    readonly format?: Format | undefined;
}) => Result.Result<string, CommandLineError>;
/**
 * Renders a command for logs and error messages.
 *
 * **When to use**
 *
 * Use when you show a command to a person, for example in an error message,
 * and secrets and control characters must not reach the output.
 *
 * **Details**
 *
 * - Quotes like {@link join}, so commands without control characters or
 *   secrets can be copied into a terminal of the given format.
 * - Replaces `Redacted` arguments with `<redacted>`.
 * - Writes control characters and Unicode line separators as escapes such as
 *   `\n` or `\u001b`, so output cannot move the cursor or inject terminal
 *   sequences.
 * - Never fails: input that {@link join} rejects is rendered anyway.
 *
 * **Gotchas**
 *
 * The output is for reading only. It is not a reliable command line: escapes
 * and `<redacted>` change what a shell would run.
 *
 * **Example** (Showing a command with a secret)
 *
 * ```ts import.meta.vitest
 * import { Redacted } from "effect"
 * import { CommandLine } from "effect/process"
 *
 * const token = Redacted.make("hunter2")
 *
 * CommandLine.display(["curl", "-H", token, "https://x.dev/a b"]) // => "curl -H <redacted> 'https://x.dev/a b'"
 * ```
 *
 * @see {@link join} for building a command line to run
 *
 * @stability unstable
 * @category encoding
 * @since 4.0.7
 */
export declare const display: (words: ReadonlyArray<Word>, options?: {
    readonly format?: Format | undefined;
}) => string;
/**
 * Builds an argument list from a template literal.
 *
 * **When to use**
 *
 * Use to write a command as it would look in a shell while keeping every
 * interpolated value a literal argument.
 *
 * **Details**
 *
 * - The template text is split into words as a POSIX shell does: spaces,
 *   tabs and line breaks separate words, and quotes and backslash escapes are
 *   removed. A long command can span several lines.
 * - Text that would need a shell fails with `UnsupportedSyntax`: operators
 *   (`| & ; < > ( )`), expansions (`$NAME`, `${...}`, `$(...)`, backticks, a
 *   leading `~`, braces such as `{a,b}`, and a leading `=` that `zsh`
 *   expands), globs (`* ? [`), comments (`#` at the start of a word), and a
 *   variable assignment in the program position. Quote such characters to
 *   pass them literally.
 * - No shell runs, so a program such as `if` or `time` names an executable.
 * - Each `${}` value is inserted literally, without splitting or quote
 *   processing. Next to other text or inside quotes, it becomes part of the
 *   surrounding argument.
 * - An array adds one argument per element and must stand alone as a whole
 *   argument.
 * - Numbers and booleans are converted with `String`.
 * - An argument that contains a `Redacted` value becomes `Redacted` as a
 *   whole.
 *
 * **Gotchas**
 *
 * - An invalid JavaScript escape sequence in the text, such as `\u` without
 *   hex digits, fails with `InvalidEscape`. Use `String.raw` or a `${}` value
 *   for backslashes.
 * - Neither the text nor the values are checked for NUL characters.
 *   {@link join} and the child process spawners reject them where they would
 *   truncate an argument.
 *
 * **Example** (Building arguments with interpolated values)
 *
 * ```ts import.meta.vitest
 * import { Redacted, Result } from "effect"
 * import { CommandLine } from "effect/process"
 *
 * const msg = "fix: handle $HOME"
 * const files = ["a.ts", "b.ts"]
 *
 * CommandLine.template`git commit -m ${msg} ${files}` // => Result.succeed(["git", "commit", "-m", msg, ...files])
 *
 * const password = Redacted.make("hunter2")
 * const words = Result.getOrThrow(CommandLine.template`psql "--password=${password}"`)
 *
 * CommandLine.display(words) // => "psql <redacted>"
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.7
 */
export declare const template: (strings: TemplateStringsArray, ...expressions: ReadonlyArray<TemplateExpression>) => Result.Result<Array<Word>, CommandLineError>;
export {};
//# sourceMappingURL=CommandLine.d.ts.map