declare const TypeId: "~effect/crypto/CryptoError";
declare const ReasonTypeId: "~effect/crypto/CryptoError/Reason";
declare const InvalidArgument_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "InvalidArgument";
} & Readonly<A>;
/**
 * Reason for an argument that the operation rejects before using the key or
 * data, such as a wrong nonce length, an out-of-range iteration count, or
 * `info` passed to a key derivation other than HKDF.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export declare class InvalidArgument extends InvalidArgument_base<{
    readonly description: string;
    readonly cause?: unknown;
}> {
    /**
     * Marks this value as a `CryptoError` reason for runtime guards.
     *
     * @since 4.0.0
     */
    readonly [ReasonTypeId]: "~effect/crypto/CryptoError/Reason";
    get message(): string;
}
declare const InvalidKey_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "InvalidKey";
} & Readonly<A>;
/**
 * Reason for key material that is rejected, such as a wrong raw key length,
 * malformed DER, PEM, or JWK data, a `CryptoKey` that does not match the typed
 * key, or a peer public key that produces an all-zero shared secret.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export declare class InvalidKey extends InvalidKey_base<{
    readonly description: string;
    readonly cause?: unknown;
}> {
    /**
     * Marks this value as a `CryptoError` reason for runtime guards.
     *
     * @since 4.0.0
     */
    readonly [ReasonTypeId]: "~effect/crypto/CryptoError/Reason";
    get message(): string;
}
declare const DecryptionFailed_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "DecryptionFailed";
} & Readonly<A>;
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
export declare class DecryptionFailed extends DecryptionFailed_base<{}> {
    /**
     * Marks this value as a `CryptoError` reason for runtime guards.
     *
     * @since 4.0.0
     */
    readonly [ReasonTypeId]: "~effect/crypto/CryptoError/Reason";
    get message(): string;
}
declare const Unsupported_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "Unsupported";
} & Readonly<A>;
/**
 * Reason for an algorithm or key that a layer does not provide, such as
 * Argon2id under `Kdf.layerWebCrypto`, or a `CryptoKey`-backed key passed to a
 * layer that cannot use it.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export declare class Unsupported extends Unsupported_base<{
    readonly description: string;
    readonly cause?: unknown;
}> {
    /**
     * Marks this value as a `CryptoError` reason for runtime guards.
     *
     * @since 4.0.0
     */
    readonly [ReasonTypeId]: "~effect/crypto/CryptoError/Reason";
    get message(): string;
}
declare const NotExtractable_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "NotExtractable";
} & Readonly<A>;
/**
 * Reason for exporting or wrapping a `CryptoKey` that was created as
 * non-extractable.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export declare class NotExtractable extends NotExtractable_base<{
    readonly description: string;
    readonly cause?: unknown;
}> {
    /**
     * Marks this value as a `CryptoError` reason for runtime guards.
     *
     * @since 4.0.0
     */
    readonly [ReasonTypeId]: "~effect/crypto/CryptoError/Reason";
    get message(): string;
}
/**
 * Union of the reasons carried by `CryptoError`.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export type CryptoErrorReason = InvalidArgument | InvalidKey | DecryptionFailed | Unsupported | NotExtractable;
declare const CryptoError_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "CryptoError";
} & Readonly<A>;
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
export declare class CryptoError extends CryptoError_base<{
    readonly module: string;
    readonly method: string;
    readonly reason: CryptoErrorReason;
}> {
    /**
     * Marks this value as a `CryptoError` for runtime guards.
     *
     * @since 4.0.0
     */
    readonly [TypeId]: "~effect/crypto/CryptoError";
    /**
     * The reason's cause, when it has one.
     *
     * @since 4.0.0
     */
    readonly cause: unknown;
    get message(): string;
}
/**
 * Checks whether a value is a `CryptoError`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export declare const isCryptoError: (u: unknown) => u is CryptoError;
/**
 * Checks whether a value is one of the reasons carried by `CryptoError`.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export declare const isCryptoErrorReason: (u: unknown) => u is CryptoErrorReason;
export {};
//# sourceMappingURL=CryptoError.d.ts.map