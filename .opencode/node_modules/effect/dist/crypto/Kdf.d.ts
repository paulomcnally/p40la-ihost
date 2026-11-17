/**
 * Key derivation: HKDF, PBKDF2, and Argon2id.
 *
 * The {@link Kdf} service derives bytes from a secret with an algorithm value
 * such as {@link hkdfSha256}, {@link pbkdf2Sha256}, or `argon2id()`. The
 * output is plain bytes: turn it into a typed key with the target family's raw
 * constructor, such as `Mac.key`.
 *
 * **Gotchas**
 *
 * - The secret is bytes, with no string overload. Encode passwords yourself,
 *   and normalize their Unicode form first if users type them on different
 *   platforms.
 * - `layerWebCrypto` does not provide Argon2id, because WebCrypto has no
 *   Argon2. Deriving with it fails with an `Unsupported` reason, so programs
 *   can fall back to PBKDF2. `NodeCrypto.layerKdf` provides Argon2id on
 *   Node.js 24.7 or later and Bun 1.4.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Context from "../Context.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
import * as CryptoError from "./CryptoError.ts";
import * as Digest from "./Digest.ts";
/**
 * HKDF (RFC 5869) with a hash, from {@link hkdf}.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface Hkdf {
    readonly _tag: "Hkdf";
    readonly name: string;
    readonly hash: Digest.Algorithm;
}
/**
 * PBKDF2 (RFC 8018) with a hash and an iteration count, from {@link pbkdf2}.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface Pbkdf2 {
    readonly _tag: "Pbkdf2";
    readonly name: string;
    readonly hash: Digest.Algorithm;
    readonly iterations: number;
}
/**
 * Argon2id (RFC 9106) with its cost parameters, from {@link argon2id}.
 *
 * **Details**
 *
 * `memory` is in KiB, `passes` is the number of passes over the memory, and
 * `parallelism` is the number of lanes.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface Argon2id {
    readonly _tag: "Argon2id";
    readonly name: string;
    readonly memory: number;
    readonly passes: number;
    readonly parallelism: number;
}
/**
 * A key derivation algorithm.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export type Algorithm = Hkdf | Pbkdf2 | Argon2id;
/**
 * Builds the HKDF algorithm for a hash.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const hkdf: (hash: Digest.Algorithm) => Hkdf;
/**
 * Builds a PBKDF2 algorithm for a hash and an iteration count.
 *
 * **Details**
 *
 * Deriving with an iteration count outside 1 to 2^31 - 1 fails with an
 * `InvalidArgument` reason.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const pbkdf2: (options: {
    readonly hash: Digest.Algorithm;
    readonly iterations: number;
}) => Pbkdf2;
/**
 * HKDF with SHA-1.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const hkdfSha1: Hkdf;
/**
 * HKDF with SHA-256.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const hkdfSha256: Hkdf;
/**
 * HKDF with SHA-384.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const hkdfSha384: Hkdf;
/**
 * HKDF with SHA-512.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const hkdfSha512: Hkdf;
/**
 * PBKDF2 with SHA-256 at 600,000 iterations, OWASP's current figure.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const pbkdf2Sha256: Pbkdf2;
/**
 * Builds an Argon2id algorithm, with `memory` in KiB.
 *
 * **Details**
 *
 * Omitted costs default to RFC 9106's second recommended setting: 64 MiB of
 * memory (`65_536`), 3 passes, and 4 lanes.
 *
 * **Gotchas**
 *
 * `Kdf.layerWebCrypto` does not provide Argon2id. Use `NodeCrypto.layerKdf`,
 * or fall back to PBKDF2 on the `Unsupported` reason.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const argon2id: (options?: {
    readonly memory?: number | undefined;
    readonly passes?: number | undefined;
    readonly parallelism?: number | undefined;
}) => Argon2id;
declare const Kdf_base: Context.ServiceClass<Kdf, "effect/crypto/Kdf", {
    /**
     * Derives `length` bytes from `secret`.
     *
     * `info` only applies to HKDF; passing it with PBKDF2 or Argon2id fails
     * with an `InvalidArgument` reason.
     */
    readonly derive: (algorithm: Algorithm, secret: Uint8Array, options: {
        readonly salt: Uint8Array;
        readonly length: number;
        readonly info?: Uint8Array | undefined;
    }) => Effect.Effect<Uint8Array, CryptoError.CryptoError>;
}>;
/**
 * Service that derives bytes from a secret.
 *
 * **Example** (Deriving a key with HKDF)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { Kdf } from "effect/crypto"
 *
 * const program = Effect.gen(function*() {
 *   const kdf = yield* Kdf.Kdf
 *   return yield* kdf.derive(Kdf.hkdfSha256, new Uint8Array(32), {
 *     salt: new Uint8Array(16),
 *     info: new TextEncoder().encode("encryption"),
 *     length: 32
 *   })
 * })
 *
 * const bytes = await Effect.runPromise(program.pipe(Effect.provide(Kdf.layerWebCrypto)))
 * bytes.length // => 32
 * ```
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare class Kdf extends Kdf_base {
}
/**
 * Layer that provides `Kdf` on the `WebCrypto` reference.
 *
 * **Details**
 *
 * HKDF and PBKDF2 run on `crypto.subtle`, off the event loop where the runtime
 * supports it. Argon2id fails with an `Unsupported` reason.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerWebCrypto: Layer.Layer<Kdf>;
export {};
//# sourceMappingURL=Kdf.d.ts.map