import * as Effect from "../../Effect.js";
import * as Predicate from "../../Predicate.js";
import * as CryptoError from "../CryptoError.js";
import { WebCrypto } from "../WebCrypto.js";
/** @internal */
export const subtleUnavailable = () => new Error("crypto.subtle is unavailable; browsers expose it only in secure contexts");
/** @internal */
export const fail = (module, method, reason) => Effect.fail(new CryptoError.CryptoError({
  module,
  method,
  reason
}));
/** @internal */
export const invalidKey = (module, method, description) => fail(module, method, new CryptoError.InvalidKey({
  description
}));
/** @internal */
export const notExtractable = (module, method) => fail(module, method, new CryptoError.NotExtractable({
  description: "The CryptoKey is not extractable"
}));
/**
 * Moves an error raised by another module's function onto `module.method`.
 *
 * @internal
 */
export const rehome = (module, method) => error => fail(module, method, error.reason);
/**
 * Thrown inside {@link run} to fail with a typed reason instead of a defect,
 * for checks that need the imported `CryptoKey`.
 *
 * @internal
 */
export class Failure {
  reason;
  constructor(reason) {
    this.reason = reason;
  }
}
/** @internal */
export const isOperationError = cause => Predicate.hasProperty(cause, "name") && cause.name === "OperationError";
/**
 * Maps an exception thrown by `crypto.subtle` in `module.layerWebCrypto` to a
 * `CryptoError`.
 *
 * A {@link Failure} carries its own reason. `DataError` and
 * `InvalidAccessError` are `InvalidKey`, and `NotSupportedError` is
 * `Unsupported`, naming the algorithm and the layer. An `OperationError`
 * becomes the reason from `onOperationError`, such as `DecryptionFailed` for a
 * decryption. Any other exception is rethrown, which `Effect.tryPromise` turns
 * into a defect.
 *
 * @internal
 */
export const webCryptoError = (module, method, algorithm, cause, onOperationError) => toCryptoError(module, method, `${module}.layerWebCrypto`, algorithm, cause, onOperationError);
const toCryptoError = (module, method, provider, algorithm, cause, onOperationError) => {
  const reason = reasonFor(method, provider, algorithm, cause, onOperationError);
  if (reason === undefined) throw cause;
  return new CryptoError.CryptoError({
    module,
    method,
    reason
  });
};
const reasonFor = (method, provider, algorithm, cause, onOperationError) => {
  if (cause instanceof Failure) return cause.reason;
  switch (Predicate.hasProperty(cause, "name") ? cause.name : undefined) {
    case "DataError":
      return new CryptoError.InvalidKey({
        description: "Invalid key data",
        cause
      });
    case "InvalidAccessError":
      return new CryptoError.InvalidKey({
        description: `The key cannot be used for ${method}`,
        cause
      });
    case "NotSupportedError":
      return new CryptoError.Unsupported({
        description: `${provider} does not provide ${algorithm}`,
        cause
      });
    case "OperationError":
      return onOperationError?.();
    default:
      return undefined;
  }
};
/**
 * Runs `f` on the `subtle` of a `Crypto` object in `module.layerWebCrypto`,
 * dying when it is missing, as in an insecure browser context. Exceptions go
 * through `onError`, which defaults to {@link webCryptoError}.
 *
 * @internal
 */
export const run = (module, method, algorithm, crypto, f, onError = cause => webCryptoError(module, method, algorithm, cause)) => Effect.suspend(() => {
  const subtle = crypto?.subtle;
  if (subtle === undefined) return Effect.die(subtleUnavailable());
  return Effect.tryPromise({
    try: () => f(subtle),
    catch: onError
  });
});
/**
 * {@link run} on the `WebCrypto` reference, for the top-level functions that
 * have no layer to read it from. `crypto.subtle` is named as the provider
 * that declined an algorithm.
 *
 * @internal
 */
export const runWithReference = (module, method, algorithm, f) => WebCrypto.use(crypto => run(module, method, algorithm, crypto, f, cause => toCryptoError(module, method, "crypto.subtle", algorithm, cause)));
//# sourceMappingURL=webCrypto.js.map