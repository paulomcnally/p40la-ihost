/**
 * Message digests: MD5, SHA-1, SHA-256, SHA-384, and SHA-512.
 *
 * The {@link Digest} service computes digests and lowercase hex strings.
 * Provide {@link layerWebCrypto} to run it on the `WebCrypto` reference,
 * or mock the service with `Layer.mock`.
 *
 * MD5 runs a pure TypeScript implementation vendored from `@noble/hashes`
 * 2.4.0, because WebCrypto does not provide it. Whether the `@noble/hashes`
 * audit covers its MD5 code has not been confirmed. The SHA algorithms run on
 * `crypto.subtle`.
 *
 * Output is stable byte for byte, so digests can be persisted.
 *
 * **Gotchas**
 *
 * MD5 and SHA-1 are broken for collision resistance. Use them only for
 * interoperability with existing protocols and checksums.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Context from "../Context.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
/**
 * A digest algorithm, such as {@link sha256}.
 *
 * **Details**
 *
 * `name` is the WebCrypto algorithm name. `implementation` is the pure
 * TypeScript code for algorithms that WebCrypto does not provide, which a
 * layer calls when it has no native mapping for `name`.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface Algorithm {
    readonly name: "MD5" | "SHA-1" | "SHA-256" | "SHA-384" | "SHA-512";
    readonly outputLength: number;
    readonly implementation: ((data: Uint8Array) => Uint8Array) | undefined;
}
/**
 * MD5 (RFC 1321), with a 16-byte output.
 *
 * **Gotchas**
 *
 * MD5 is broken for collision resistance. Use it only for interoperability
 * with existing protocols and checksums.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const md5: Algorithm;
/**
 * SHA-1, with a 20-byte output.
 *
 * **Gotchas**
 *
 * SHA-1 is broken for collision resistance. Use it only for interoperability
 * with existing protocols.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const sha1: Algorithm;
/**
 * SHA-256, with a 32-byte output.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const sha256: Algorithm;
/**
 * SHA-384, with a 48-byte output.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const sha384: Algorithm;
/**
 * SHA-512, with a 64-byte output.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export declare const sha512: Algorithm;
declare const Digest_base: Context.ServiceClass<Digest, "effect/crypto/Digest", {
    /**
     * Computes the digest of `data`.
     */
    readonly digest: (algorithm: Algorithm, data: Uint8Array) => Effect.Effect<Uint8Array>;
    /**
     * Computes the digest of `data` as lowercase hex.
     */
    readonly digestHex: (algorithm: Algorithm, data: Uint8Array) => Effect.Effect<string>;
}>;
/**
 * Service that computes message digests.
 *
 * **Example** (Hashing with SHA-256)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { Digest } from "effect/crypto"
 *
 * const program = Effect.gen(function*() {
 *   const digest = yield* Digest.Digest
 *   return yield* digest.digestHex(Digest.sha256, new TextEncoder().encode("abc"))
 * })
 *
 * const hex = await Effect.runPromise(program.pipe(Effect.provide(Digest.layerWebCrypto)))
 * hex.length // => 64
 * ```
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare class Digest extends Digest_base {
}
/**
 * Layer that provides `Digest` on the `WebCrypto` reference.
 *
 * **Details**
 *
 * The SHA algorithms run on `crypto.subtle`, and MD5 on the pure TypeScript
 * implementation. Without `crypto.subtle`, the layer still builds and MD5
 * keeps working, but SHA digests die.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerWebCrypto: Layer.Layer<Digest>;
export {};
//# sourceMappingURL=Digest.d.ts.map