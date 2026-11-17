import * as Bash from "./internal/completions/bash.js";
import * as Descriptor from "./internal/completions/descriptor.js";
import * as Fish from "./internal/completions/fish.js";
import * as PowerShell from "./internal/completions/powershell.js";
import * as Zsh from "./internal/completions/zsh.js";
/**
 * Converts a command tree into a `CommandDescriptor` for completion script
 * generation.
 *
 * **When to use**
 *
 * Use when you need to generate a completion script from an existing
 * `Command`, for example in a build step, without running the CLI or
 * capturing the output of the built-in `--completions` flag.
 *
 * **Details**
 *
 * Reads the command's flags, arguments, and subcommands, including flags
 * shared with subcommands and aliases. Hidden flags and unlisted subcommands
 * are omitted, matching `--help`. Command handlers are never run.
 *
 * **Example** (Generating a Zsh completion script at build time)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { Command, Completions, Flag } from "effect/cli"
 *
 * const start = Command.make(
 *   "start",
 *   { port: Flag.Int("port") },
 *   ({ port }) => Effect.log(`Listening on ${port}`)
 * )
 *
 * const cli = Command.make("server").pipe(
 *   Command.withSubcommands([start])
 * )
 *
 * const script = Completions.generate(
 *   "server",
 *   "zsh",
 *   Completions.fromCommand(cli)
 * )
 *
 * script.startsWith("#compdef server") // => true
 * ```
 *
 * @see {@link generate} for turning the descriptor into a shell script
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.7
 */
export const fromCommand = command => Descriptor.fromCommand(command);
/**
 * Generates a shell completion script for a command descriptor.
 *
 * **When to use**
 *
 * Use when you need an installable completion script from an existing
 * `CommandDescriptor`.
 *
 * **Details**
 *
 * Dispatches by `shell` to Bash, Zsh, Fish, or PowerShell generation and
 * returns a static script string for `executableName`.
 *
 * @see {@link Shell} for supported shell names
 * @see {@link CommandDescriptor} for the command shape used by completion generation
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const generate = (executableName, shell, descriptor) => {
  switch (shell) {
    case "bash":
      return Bash.generate(executableName, descriptor);
    case "zsh":
      return Zsh.generate(executableName, descriptor);
    case "fish":
      return Fish.generate(executableName, descriptor);
    case "powershell":
      return PowerShell.generate(executableName, descriptor);
  }
};
//# sourceMappingURL=Completions.js.map