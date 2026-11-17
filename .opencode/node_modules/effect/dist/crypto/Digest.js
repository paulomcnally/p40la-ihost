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
import * as Context from "../Context.js";
import * as Effect from "../Effect.js";
import * as Hex from "../encoding/Hex.js";
import * as Layer from "../Layer.js";
import { webCryptoDigest } from "./internal/digest.js";
import * as Md5 from "./internal/md5.js";
import { WebCrypto } from "./WebCrypto.js";
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
export const md5 = {
  name: "MD5",
  outputLength: 16,
  implementation: Md5.md5
};
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
export const sha1 = {
  name: "SHA-1",
  outputLength: 20,
  implementation: undefined
};
/**
 * SHA-256, with a 32-byte output.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const sha256 = {
  name: "SHA-256",
  outputLength: 32,
  implementation: undefined
};
/**
 * SHA-384, with a 48-byte output.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const sha384 = {
  name: "SHA-384",
  outputLength: 48,
  implementation: undefined
};
/**
 * SHA-512, with a 64-byte output.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const sha512 = {
  name: "SHA-512",
  outputLength: 64,
  implementation: undefined
};
/**
 * Converts a hash name to an algorithm-name suffix: `"SHA-256"` to `"Sha256"`.
 *
 * @internal
 */
export const nameSuffix = hash => hash.name.charAt(0) + hash.name.slice(1).toLowerCase().replace("-", "");
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
export class Digest extends /*#__PURE__*/Context.Service()("effect/crypto/Digest") {}
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
export const layerWebCrypto = /*#__PURE__*/Layer.effect(Digest, /*#__PURE__*/WebCrypto.use(crypto => Effect.succeed({
  digest: (algorithm, data) => webCryptoDigest(crypto, algorithm, data),
  digestHex: (algorithm, data) => Effect.map(webCryptoDigest(crypto, algorithm, data), Hex.encode)
})));
//# sourceMappingURL=Digest.js.map