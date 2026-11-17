import * as Effect from "../../Effect.js";
import * as internal from "./webCrypto.js";
/** @internal */
export const webCryptoDigest = (crypto, algorithm, data) => {
  const implementation = algorithm.implementation;
  if (implementation !== undefined) {
    return Effect.sync(() => implementation(data));
  }
  return Effect.suspend(() => {
    const subtle = crypto?.subtle;
    if (subtle === undefined) return Effect.die(internal.subtleUnavailable());
    return Effect.promise(async () => new Uint8Array(await subtle.digest(algorithm.name, data)));
  });
};
//# sourceMappingURL=digest.js.map