/**
 * Parses TOML configuration files.
 *
 * The implementation covers the TOML values and table forms used by Effect's
 * configuration-file CLI primitive while avoiding a generated parser runtime.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Result from "../Result.ts";
import { EncodingError } from "./EncodingError.ts";
/**
 * Parses a TOML document into a null-prototype record.
 *
 * **Details**
 *
 * Returns `Result.succeed` with the parsed record, or `Result.fail` with an
 * `EncodingError` when the input is not valid TOML. Offset date-times are
 * represented by `Date`, matching the package this parser replaces. Local
 * dates and times remain strings.
 *
 * **Example** (Parsing a TOML document)
 *
 * ```ts import.meta.vitest
 * import { Result } from "effect"
 * import { Toml } from "effect/encoding"
 *
 * Toml.parse("port = 8080") // => Result.succeed({ port: 8080 })
 * Result.isFailure(Toml.parse("port = ")) // => true
 * ```
 *
 * @stability unstable
 * @category decoding
 * @since 4.0.0
 */
export declare const parse: (input: string) => Result.Result<Record<string, unknown>, EncodingError>;
//# sourceMappingURL=Toml.d.ts.map