/**
 * The `CryptoError` returned by the `effect/crypto` modules, and the reasons it
 * carries.
 *
 * Every failure is one `CryptoError` whose `module` and `method` name the
 * operation, such as `"Aead"` and `"decrypt"`, and whose `reason` says what
 * went wrong: `InvalidArgument`, `InvalidKey`, `DecryptionFailed`,
 * `Unsupported`, or `NotExtractable`.
 *
 * **Gotchas**
 *
 * The error is not serializable, so it cannot be a typed failure in an Rpc or
 * HttpApi schema. Map it before it reaches a client, for example
 * `DecryptionFailed` to a 401. Sending crypto failure detail to clients is how
 * oracles leak.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Data from "../Data.js";
import * as Predicate from "../Predicate.js";
const TypeId = "~effect/crypto/CryptoError";
const ReasonTypeId = "~effect/crypto/CryptoError/Reason";
/**
 * Reason for an argument that the operation rejects before using the key or
 * data, such as a wrong nonce length, an out-of-range iteration count, or
 * `info` passed to a key derivation other than HKDF.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class InvalidArgument extends /*#__PURE__*/Data.TaggedError("InvalidArgument") {
  /**
   * Marks this value as a `CryptoError` reason for runtime guards.
   *
   * @since 4.0.0
   */
  [ReasonTypeId] = ReasonTypeId;
  get message() {
    return this.description;
  }
}
/**
 * Reason for key material that is rejected, such as a wrong raw key length,
 * malformed DER, PEM, or JWK data, a `CryptoKey` that does not match the typed
 * key, or a peer public key that produces an all-zero shared secret.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class InvalidKey extends /*#__PURE__*/Data.TaggedError("InvalidKey") {
  /**
   * Marks this value as a `CryptoError` reason for runtime guards.
   *
   * @since 4.0.0
   */
  [ReasonTypeId] = ReasonTypeId;
  get message() {
    return this.description;
  }
}
/**
 * Reason for a decryption or unwrap that failed its integrity check, or a CBC
 * ciphertext with bad padding or length.
 *
 * **Details**
 *
 * The reason is identical for every cause and carries no `cause` or runtime
 * message, so callers cannot tell a tampered tag from bad padding. Keeping
 * either would create a padding oracle.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class DecryptionFailed extends /*#__PURE__*/Data.TaggedError("DecryptionFailed") {
  /**
   * Marks this value as a `CryptoError` reason for runtime guards.
   *
   * @since 4.0.0
   */
  [ReasonTypeId] = ReasonTypeId;
  get message() {
    return "Could not decrypt the data";
  }
}
/**
 * Reason for an algorithm or key that a layer does not provide, such as
 * Argon2id under `Kdf.layerWebCrypto`, or a `CryptoKey`-backed key passed to a
 * layer that cannot use it.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class Unsupported extends /*#__PURE__*/Data.TaggedError("Unsupported") {
  /**
   * Marks this value as a `CryptoError` reason for runtime guards.
   *
   * @since 4.0.0
   */
  [ReasonTypeId] = ReasonTypeId;
  get message() {
    return this.description;
  }
}
/**
 * Reason for exporting or wrapping a `CryptoKey` that was created as
 * non-extractable.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class NotExtractable extends /*#__PURE__*/Data.TaggedError("NotExtractable") {
  /**
   * Marks this value as a `CryptoError` reason for runtime guards.
   *
   * @since 4.0.0
   */
  [ReasonTypeId] = ReasonTypeId;
  get message() {
    return this.description;
  }
}
/**
 * Error returned by the `effect/crypto` modules.
 *
 * **When to use**
 *
 * Use to handle failures from any `effect/crypto` operation, matching on
 * `reason` to separate caller mistakes from failed decryption.
 *
 * **Details**
 *
 * `module` and `method` name the operation, such as `"Aead"` and `"decrypt"`.
 * The message reads `"Aead.decrypt: <reason message>"`. The error's `cause`
 * is the reason's `cause`, when it has one.
 *
 * **Example** (Recovering from failed decryption)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { CryptoError } from "effect/crypto"
 *
 * const decrypt = Effect.fail(
 *   new CryptoError.CryptoError({
 *     module: "Aead",
 *     method: "decrypt",
 *     reason: new CryptoError.DecryptionFailed()
 *   })
 * )
 *
 * const program = decrypt.pipe(
 *   Effect.catchReason("CryptoError", "DecryptionFailed", () => Effect.succeed("rejected"))
 * )
 *
 * await Effect.runPromise(program) // => "rejected"
 * ```
 *
 * @see {@link isCryptoError} for narrowing unknown values
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class CryptoError extends /*#__PURE__*/Data.TaggedError("CryptoError") {
  /**
   * Marks this value as a `CryptoError` for runtime guards.
   *
   * @since 4.0.0
   */
  [TypeId] = TypeId;
  /**
   * The reason's cause, when it has one.
   *
   * @since 4.0.0
   */
  cause = "cause" in this.reason ? this.reason.cause : undefined;
  get message() {
    return `${this.module}.${this.method}: ${this.reason.message}`;
  }
}
/**
 * Checks whether a value is a `CryptoError`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export const isCryptoError = u => Predicate.hasProperty(u, TypeId);
/**
 * Checks whether a value is one of the reasons carried by `CryptoError`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export const isCryptoErrorReason = u => Predicate.hasProperty(u, ReasonTypeId);
//# sourceMappingURL=CryptoError.js.map