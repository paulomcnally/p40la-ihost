/**
 * Describes child processes before they are started.
 *
 * A `Command` stores the executable, arguments, environment, standard streams,
 * working directory, and other process options. Commands can also be piped
 * together. A command is an `Effect`; running it asks the
 * `ChildProcessSpawner` service to start the process and returns a
 * `ChildProcessHandle`.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Context from "../Context.js";
import * as Effectable from "../Effectable.js";
import { dual } from "../Function.js";
import * as Predicate from "../Predicate.js";
import * as Result from "../Result.js";
import { ChildProcessSpawner } from "./ChildProcessSpawner.js";
import * as CommandLine from "./CommandLine.js";
const TypeId = "~effect/process/ChildProcess";
// =============================================================================
// Constructors
// =============================================================================
const Proto = {
  ... /*#__PURE__*/Effectable.Prototype({
    label: "Command",
    evaluate(fiber) {
      return Context.getUnsafe(fiber.context, ChildProcessSpawner).spawn(this);
    }
  }),
  [TypeId]: TypeId
};
/**
 * Checks whether a value is a `Command`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export const isCommand = u => Predicate.hasProperty(u, TypeId);
/**
 * Checks whether a command is a `StandardCommand`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export const isStandardCommand = command => command._tag === "StandardCommand";
/**
 * Checks whether a command is a `PipedCommand`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export const isPipedCommand = command => command._tag === "PipedCommand";
const makeStandardCommand = (command, args, options) => Object.assign(Object.create(Proto), {
  _tag: "StandardCommand",
  command,
  args,
  options
});
const makePipedCommand = (left, right, options = {}) => Object.assign(Object.create(Proto), {
  _tag: "PipedCommand",
  left,
  right,
  options
});
/**
 * Create a command from a template literal, options + template, or array form.
 *
 * **Details**
 *
 * This function supports three calling conventions:
 * 1. Template literal: `make\`npm run build\``
 * 2. Options + template literal: `make({ cwd: "/app" })\`npm run build\``
 * 3. Array form: `make("npm", ["run", "build"], options?)`
 *
 * Templates are split with `CommandLine.template`: quotes group words, each
 * `${}` value is one literal argument, and an argument that contains a
 * `Redacted` value stays `Redacted`. Arguments in the array form can be
 * `Redacted` too.
 *
 * With the `shell` option, the template text is the shell script and is kept
 * as written.
 *
 * **Gotchas**
 *
 * - A template is parsed when `make` is called. Shell syntax such as `|`,
 *   `$HOME`, `*` or `~` throws a `CommandLine.CommandLineError`: quote it, or
 *   pass the value with `${}`.
 * - A template with the `shell` option cannot contain `${}` values, because
 *   their quoting depends on the shell. Build the script with
 *   `CommandLine.quote` and pass it as a string.
 * - The program name, the first word, cannot be `Redacted`.
 * - A NUL character in the command, an argument or an `env` value fails with
 *   a `BadArgument` error when the command is spawned.
 *
 * **Example** (Creating commands)
 *
 * ```ts import.meta.vitest
 * import { ChildProcess } from "effect/process"
 *
 * // Template literal form; quotes group words
 * const cmd1 = ChildProcess.make`echo "hello world"`
 *
 * // With options
 * const cmd2 = ChildProcess.make({ cwd: "/tmp" })`ls -la`
 *
 * // Array form
 * const cmd3 = ChildProcess.make("git", ["status"])
 *
 * const result = [cmd1.args[0], cmd2.options.cwd, cmd3.args[0]] // => ["hello world", "/tmp", "status"]
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const make = function make(...args) {
  // Template literal form: make`command`
  if (isTemplateString(args[0])) {
    const [templates, ...expressions] = args;
    const [command, cmdArgs] = parseTemplate(templates, expressions);
    return makeStandardCommand(command, cmdArgs, {});
  }
  // Options form: make({ cwd: "/tmp" })`command`
  if (typeof args[0] === "object" && !Array.isArray(args[0]) && !isTemplateString(args[0])) {
    const options = args[0];
    return function (templates, ...expressions) {
      if (options.shell !== undefined && options.shell !== false) {
        return makeStandardCommand(shellScript(templates, expressions), [], options);
      }
      const [command, cmdArgs] = parseTemplate(templates, expressions);
      return makeStandardCommand(command, cmdArgs, options);
    };
  }
  // Standard form without arguments: make("command", options?)
  if (typeof args[0] === "string" && !Array.isArray(args[1])) {
    const [command, options = {}] = args;
    return makeStandardCommand(command, [], options);
  }
  // Standard form with arguments: make("command", ["arg1", "arg2"], options?)
  const [command, cmdArgs = [], options = {}] = args;
  return makeStandardCommand(command, cmdArgs, options);
};
/**
 * Pipes the output of one command to the input of another.
 *
 * **Details**
 *
 * By default, pipes `stdout` from the source to `stdin` of the destination.
 * Use the `options` parameter to customize which streams are connected.
 *
 * **Example** (Piping command output)
 *
 * ```ts import.meta.vitest
 * import { ChildProcess } from "effect/process"
 *
 * // Pipe stdout (default)
 * const pipeline1 = ChildProcess.make`cat file.txt`.pipe(
 *   ChildProcess.pipeTo(ChildProcess.make`grep pattern`)
 * )
 *
 * // Pipe stderr instead of stdout
 * const pipeline2 = ChildProcess.make`my-program`.pipe(
 *   ChildProcess.pipeTo(ChildProcess.make`grep error`, { from: "stderr" })
 * )
 *
 * // Pipe combined stdout and stderr
 * const pipeline3 = ChildProcess.make`my-program`.pipe(
 *   ChildProcess.pipeTo(ChildProcess.make`tee output.log`, { from: "all" })
 * )
 *
 * const result = [pipeline1._tag, pipeline2.options.from, pipeline3.options.from]
 * result // => ["PipedCommand", "stderr", "all"]
 * ```
 *
 * @stability unstable
 * @category combinators
 * @since 4.0.0
 */
export const pipeTo = /*#__PURE__*/dual(args => isCommand(args[0]) && isCommand(args[1]), (self, that, options) => makePipedCommand(self, that, options ?? {}));
/**
 * Prepends another command to a command.
 *
 * **Details**
 *
 * For pipelines, only the leftmost command is prefixed.
 *
 * **Example** (Prefixing commands)
 *
 * ```ts import.meta.vitest
 * import { ChildProcess } from "effect/process"
 *
 * const command = ChildProcess.make`echo foo`
 *
 * const prefixed = command.pipe(
 *   ChildProcess.prefix`time`
 * )
 *
 * // now prefixed will execute `time echo "foo"`
 * const result = prefixed._tag === "StandardCommand" ? `${prefixed.command} ${prefixed.args[0]}` : prefixed._tag
 * result // => "time echo"
 * ```
 *
 * @stability unstable
 * @category combinators
 * @since 4.0.0
 */
export const prefix = function prefix(...args) {
  if (isCommand(args[0]) && args.length > 1) {
    const [self, ...rest] = args;
    const prefixSpec = parsePrefixArgs(rest);
    return applyPrefix(self, prefixSpec);
  }
  const prefixSpec = parsePrefixArgs(args);
  return self => applyPrefix(self, prefixSpec);
};
const parsePrefixArgs = args => {
  if (isTemplateString(args[0])) {
    const [templates, ...expressions] = args;
    const [command, cmdArgs] = parseTemplate(templates, expressions);
    return {
      command,
      args: cmdArgs
    };
  }
  const [command, cmdArgs = []] = args;
  return {
    command,
    args: cmdArgs
  };
};
const applyPrefix = (self, prefixSpec) => {
  switch (self._tag) {
    case "StandardCommand":
      {
        return makeStandardCommand(prefixSpec.command, [...prefixSpec.args, self.command, ...self.args], self.options);
      }
    case "PipedCommand":
      {
        return makePipedCommand(applyPrefix(self.left, prefixSpec), self.right, self.options);
      }
  }
};
/**
 * Sets the current working directory for a command.
 *
 * **Details**
 *
 * For pipelines, applies to each command in the pipeline.
 *
 * **Example** (Setting command working directories)
 *
 * ```ts import.meta.vitest
 * import { ChildProcess } from "effect/process"
 *
 * const cmd = ChildProcess.make`ls -la`.pipe(
 *   ChildProcess.setCwd("/tmp")
 * )
 * const result = cmd._tag === "StandardCommand" && cmd.options.cwd // => "/tmp"
 * ```
 *
 * @stability unstable
 * @category combinators
 * @since 4.0.0
 */
export const setCwd = /*#__PURE__*/dual(2, (self, cwd) => {
  switch (self._tag) {
    case "StandardCommand":
      {
        return makeStandardCommand(self.command, self.args, {
          ...self.options,
          cwd
        });
      }
    case "PipedCommand":
      {
        return makePipedCommand(setCwd(self.left, cwd), setCwd(self.right, cwd), self.options);
      }
  }
});
/**
 * Adds environment variables to a command, merging them with any existing
 * command environment and overriding duplicate keys.
 *
 * **Details**
 *
 * For pipelines, applies to each command in the pipeline.
 *
 * **Example** (Setting command environment variables)
 *
 * ```ts import.meta.vitest
 * import { ChildProcess } from "effect/process"
 *
 * const cmd = ChildProcess.make`node script.js`.pipe(
 *   ChildProcess.setEnv({ NODE_ENV: "test" })
 * )
 * const result = cmd._tag === "StandardCommand" && cmd.options.env?.NODE_ENV // => "test"
 * ```
 *
 * @stability unstable
 * @category combinators
 * @since 4.0.0
 */
export const setEnv = /*#__PURE__*/dual(2, (self, env) => {
  switch (self._tag) {
    case "StandardCommand":
      {
        const nextEnv = self.options.env === undefined ? env : {
          ...self.options.env,
          ...env
        };
        return makeStandardCommand(self.command, self.args, {
          ...self.options,
          env: nextEnv
        });
      }
    case "PipedCommand":
      {
        return makePipedCommand(setEnv(self.left, env), setEnv(self.right, env), self.options);
      }
  }
});
const isTemplateString = u => Array.isArray(u) && "raw" in u && Array.isArray(u.raw);
// =============================================================================
// Utilities
// =============================================================================
/**
 * Parses an fd name like "fd3" to its numeric index.
 * Returns undefined if the name is invalid.
 *
 * @stability unstable
 * @category converting
 * @since 4.0.0
 */
export const parseFdName = name => {
  const match = /^fd(\d+)$/.exec(name);
  if (match === null) return undefined;
  const fd = parseInt(match[1], 10);
  return fd >= 3 ? fd : undefined;
};
/**
 * Create an fd name from its numeric index.
 *
 * @stability unstable
 * @category converting
 * @since 4.0.0
 */
export const fdName = fd => `fd${fd}`;
// =============================================================================
// Template Parsing
// =============================================================================
const parseTemplate = (templates, expressions) => {
  const [program = "", ...args] = Result.getOrThrow(CommandLine.template(templates, ...expressions));
  if (typeof program !== "string") {
    throw new CommandLine.CommandLineError({
      reason: "InvalidArgument",
      message: "The program name of a command cannot be Redacted"
    });
  }
  return [program, args];
};
const shellScript = (templates, expressions) => {
  if (expressions.length > 0) {
    throw new CommandLine.CommandLineError({
      reason: "UnsupportedSyntax",
      message: "A shell template cannot interpolate values; build the script with CommandLine.quote"
    });
  }
  const script = templates[0];
  if (script === undefined) {
    throw new CommandLine.CommandLineError({
      reason: "InvalidEscape",
      message: "Template part 0 contains an invalid escape sequence"
    });
  }
  return script;
};
//# sourceMappingURL=ChildProcess.js.map