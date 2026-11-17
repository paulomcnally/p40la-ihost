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
import * as Context from "../Context.js";
import * as Effect from "../Effect.js";
import * as Layer from "../Layer.js";
import * as CryptoError from "./CryptoError.js";
import * as Digest from "./Digest.js";
import * as internal from "./internal/webCrypto.js";
import { WebCrypto } from "./WebCrypto.js";
/**
 * Builds the HKDF algorithm for a hash.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const hkdf = hash => ({
  _tag: "Hkdf",
  name: `hkdf${Digest.nameSuffix(hash)}`,
  hash
});
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
export const pbkdf2 = options => ({
  _tag: "Pbkdf2",
  name: `pbkdf2${Digest.nameSuffix(options.hash)}`,
  hash: options.hash,
  iterations: options.iterations
});
/**
 * HKDF with SHA-1.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const hkdfSha1 = /*#__PURE__*/hkdf(Digest.sha1);
/**
 * HKDF with SHA-256.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const hkdfSha256 = /*#__PURE__*/hkdf(Digest.sha256);
/**
 * HKDF with SHA-384.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const hkdfSha384 = /*#__PURE__*/hkdf(Digest.sha384);
/**
 * HKDF with SHA-512.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const hkdfSha512 = /*#__PURE__*/hkdf(Digest.sha512);
/**
 * PBKDF2 with SHA-256 at 600,000 iterations, OWASP's current figure.
 *
 * @stability unstable
 * @category algorithms
 * @since 4.0.0
 */
export const pbkdf2Sha256 = /*#__PURE__*/pbkdf2({
  hash: Digest.sha256,
  iterations: 600_000
});
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
export const argon2id = options => ({
  _tag: "Argon2id",
  name: "argon2id",
  memory: options?.memory ?? 65_536,
  passes: options?.passes ?? 3,
  parallelism: options?.parallelism ?? 4
});
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
export class Kdf extends /*#__PURE__*/Context.Service()("effect/crypto/Kdf") {}
const invalidArgument = description => internal.fail("Kdf", "derive", new CryptoError.InvalidArgument({
  description
}));
const isIntegerIn = (n, min, max) => Number.isSafeInteger(n) && n >= min && n <= max;
// Checks the arguments before they reach Web Crypto, which takes a 32-bit bit
// length that some backends wrap instead of rejecting. Node.js aborts the
// process on PBKDF2 iteration counts above 2^31 - 1.
const checkArguments = (algorithm, options) => {
  if (algorithm._tag !== "Hkdf" && options.info !== undefined) return invalidArgument("info only applies to HKDF");
  if (algorithm._tag === "Argon2id") return undefined;
  if (algorithm.hash.name === "MD5") return invalidArgument(`${algorithm._tag} needs a SHA hash, not MD5`);
  if (algorithm._tag === "Hkdf") {
    const maxLength = 255 * algorithm.hash.outputLength;
    return isIntegerIn(options.length, 1, maxLength) ? undefined : invalidArgument(`length must be between 1 and ${maxLength} bytes for ${algorithm.name}`);
  }
  if (!isIntegerIn(algorithm.iterations, 1, 0x7fff_ffff)) {
    return invalidArgument("iterations must be between 1 and 2^31 - 1");
  }
  return isIntegerIn(options.length, 1, 0x1fff_ffff) ? undefined : invalidArgument("length must be between 1 and 2^29 - 1 bytes");
};
const webCryptoParams = (algorithm, options) => algorithm._tag === "Hkdf" ? {
  name: "HKDF",
  hash: algorithm.hash.name,
  salt: options.salt,
  info: options.info ?? new Uint8Array()
} : {
  name: "PBKDF2",
  hash: algorithm.hash.name,
  salt: options.salt,
  iterations: algorithm.iterations
};
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
export const layerWebCrypto = /*#__PURE__*/Layer.effect(Kdf, /*#__PURE__*/WebCrypto.use(crypto => Effect.succeed(Kdf.of({
  derive: (algorithm, secret, options) => Effect.suspend(() => {
    const failure = checkArguments(algorithm, options);
    if (failure !== undefined) return failure;
    if (algorithm._tag === "Argon2id") {
      return internal.fail("Kdf", "derive", new CryptoError.Unsupported({
        description: "Kdf.layerWebCrypto does not provide argon2id"
      }));
    }
    const name = algorithm._tag === "Hkdf" ? "HKDF" : "PBKDF2";
    return internal.run("Kdf", "derive", algorithm.name, crypto, async subtle => {
      const handle = await subtle.importKey("raw", secret, name, false, ["deriveBits"]);
      return new Uint8Array(await subtle.deriveBits(webCryptoParams(algorithm, options), handle, options.length * 8));
    });
  })
}))));
//# sourceMappingURL=Kdf.js.map