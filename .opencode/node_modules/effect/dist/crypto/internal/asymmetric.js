import * as Effect from "../../Effect.js";
import * as Predicate from "../../Predicate.js";
import * as CryptoError from "../CryptoError.js";
import * as KeyModel from "./key.js";
import * as internal from "./webCrypto.js";
// RFC 8410: the PKCS#8 encoding of a 32-byte Ed25519 or X25519 private key is
// this prefix, ending in the algorithm's OID byte, followed by the key.
const seedPrefix = oid => Uint8Array.of(48, 46, 2, 1, 0, 48, 5, 6, 3, 43, 101, oid, 4, 34, 4, 32);
/** @internal */
export const ed25519Prefix = /*#__PURE__*/seedPrefix(112);
/** @internal */
export const x25519Prefix = /*#__PURE__*/seedPrefix(110);
const seedLength = 32;
/**
 * Coordinate sizes in bytes, which are also the sizes of an ECDSA signature's
 * `r` and `s` and of an ECDH shared secret.
 *
 * @internal
 */
export const coordinateSizes = {
  "P-256": 32,
  "P-384": 48,
  "P-521": 66
};
/**
 * Checks raw Ed25519 or X25519 key length, with guidance for 64-byte seeds.
 *
 * @internal
 */
export const checkSeedLength = (name, role) => bytes => {
  if (bytes.length === seedLength) return undefined;
  if (role === "private" && bytes.length === 64) {
    return `Expected the 32-byte ${name} seed, not the 64-byte seed‖public key form: pass its first 32 bytes`;
  }
  return `Expected a ${seedLength}-byte ${name} ${role} key, not ${bytes.length} bytes`;
};
/**
 * Checks the encoding of a raw EC public key: an uncompressed SEC1 point,
 * `0x04` followed by both coordinates. Whether the point is on the curve is
 * checked when the key is first used.
 *
 * @internal
 */
export const checkPoint = curve => bytes => {
  const length = 1 + 2 * coordinateSizes[curve];
  if (bytes.length !== length) return `Expected a ${length}-byte uncompressed ${curve} point, not ${bytes.length} bytes`;
  return bytes[0] === 4 ? undefined : `Expected an uncompressed ${curve} point, starting with 0x04`;
};
/** @internal */
export const checkNamedCurve = curve => algorithm => {
  const namedCurve = algorithm.namedCurve;
  return namedCurve === curve ? undefined : `Expected a ${curve} key, not ${namedCurve}`;
};
/** @internal */
export const checkHash = hash => algorithm => {
  const keyHash = algorithm.hash?.name;
  return keyHash === hash ? undefined : `Expected a key for ${hash}, not ${keyHash}`;
};
/** @internal */
export const rsaGenerateParams = (spec, modulusLength) => ({
  ...spec.importParams,
  modulusLength,
  publicExponent: Uint8Array.of(1, 0, 1)
});
/**
 * Synchronously validates and wraps a `CryptoKey`.
 *
 * @internal
 */
export const fromCryptoKey = (family, method, algorithm, role, cryptoKey) => {
  const spec = family.specOf(algorithm);
  return KeyModel.fromCryptoKey(family.module, method, role === "private" ? family.privateKeyProto : family.publicKeyProto, algorithm, cryptoKey, {
    type: role,
    name: spec.name,
    usages: role === "private" ? spec.privateUsages : spec.publicUsages,
    usagePolicy: "all",
    check: spec.check
  });
};
const concat = (prefix, bytes) => {
  const out = new Uint8Array(prefix.length + bytes.length);
  out.set(prefix);
  out.set(bytes, prefix.length);
  return out;
};
/** @internal */
export const importPrivate = (subtle, bytes, spec, extractable) => subtle.importKey("pkcs8", spec.seedPrefix === undefined ? bytes : concat(spec.seedPrefix, bytes), spec.importParams, extractable, spec.privateUsages);
/** @internal */
export const importPublic = (subtle, bytes, spec) => subtle.importKey(spec.publicFormat, bytes, spec.importParams, true, spec.publicUsages);
/**
 * Exports a private `CryptoKey` to its stored form: the seed for Ed25519 and
 * X25519, PKCS#8 otherwise. A seed export in another PKCS#8 form is one the
 * layer cannot read, so it fails with `Unsupported`.
 */
const exportPrivate = async (subtle, cryptoKey, spec) => {
  const pkcs8 = new Uint8Array(await subtle.exportKey("pkcs8", cryptoKey));
  const prefix = spec.seedPrefix;
  if (prefix === undefined) return pkcs8;
  if (pkcs8.length !== prefix.length + seedLength || !prefix.every((byte, i) => pkcs8[i] === byte)) {
    throw new internal.Failure(new CryptoError.Unsupported({
      description: `${spec.name} private key was exported in an unexpected PKCS#8 form`
    }));
  }
  return pkcs8.slice(prefix.length);
};
const exportPublic = async (subtle, cryptoKey, spec) => new Uint8Array(await subtle.exportKey(spec.publicFormat, cryptoKey));
/** @internal */
export const exportKey = (subtle, cryptoKey, spec, role) => role === "private" ? exportPrivate(subtle, cryptoKey, spec) : exportPublic(subtle, cryptoKey, spec);
/**
 * Exports a key's stored representation as fresh bytes, reading the
 * `WebCrypto` reference for `CryptoKey`-backed keys.
 *
 * @internal
 */
export const toBytes = (family, key) => KeyModel.toBytes(family.module, key, (subtle, cryptoKey) => exportKey(subtle, cryptoKey, family.specOf(key.algorithm), family.isPrivateKey(key) ? "private" : "public"));
/**
 * Per-layer caches for imported byte-backed keys.
 *
 * @internal
 */
export const makeCaches = family => ({
  privateKey: KeyModel.makeCache((subtle, bytes, algorithm) => importPrivate(subtle, bytes, family.specOf(algorithm), false)),
  publicKey: KeyModel.makeCache((subtle, bytes, algorithm) => importPublic(subtle, bytes, family.specOf(algorithm)))
});
/**
 * Computes the stored public key of an extractable private `CryptoKey`.
 *
 * WebCrypto cannot derive a public key directly, but a private key's JWK
 * export carries the public members, which import as a public key.
 */
const publicOf = async (subtle, privateKey, spec) => {
  const {
    d,
    p,
    q,
    dp,
    dq,
    qi,
    oth,
    key_ops,
    ext,
    alg,
    ...jwk
  } = await subtle.exportKey("jwk", privateKey);
  const publicKey = await subtle.importKey("jwk", jwk, spec.importParams, true, spec.publicUsages);
  return exportPublic(subtle, publicKey, spec);
};
/**
 * Generates a key pair in its stored forms. Ed25519 and X25519 private keys
 * are 32 bytes from `SecureRandom`, so a seeded `SecureRandom` reproduces
 * them. EC and RSA keys come from WebCrypto's own randomness.
 *
 * @internal
 */
export const generateKeyPair = (family, crypto, random, algorithm, params) => {
  const spec = family.specOf(algorithm);
  const generate = async (subtle, seed) => {
    if (seed !== undefined) {
      const privateKey = await importPrivate(subtle, seed, spec, true);
      return {
        privateKey: seed,
        publicKey: await publicOf(subtle, privateKey, spec)
      };
    }
    const usages = [...new Set([...spec.privateUsages, ...spec.publicUsages])];
    const pair = await subtle.generateKey(params, true, usages);
    return {
      privateKey: await exportPrivate(subtle, pair.privateKey, spec),
      publicKey: await exportPublic(subtle, pair.publicKey, spec)
    };
  };
  return Effect.flatMap(spec.seedPrefix === undefined ? Effect.undefined : random.randomBytes(seedLength), seed => Effect.map(internal.run(family.module, "generateKeyPair", algorithm.name, crypto, subtle => generate(subtle, seed)), ({
    privateKey,
    publicKey
  }) => ({
    privateKey: KeyModel.makeBytes(family.privateKeyProto, algorithm, privateKey),
    publicKey: KeyModel.makeBytes(family.publicKeyProto, algorithm, publicKey)
  })));
};
/**
 * Computes the public key of a private key, failing with `NotExtractable`
 * for a non-extractable `CryptoKey`, whose public members cannot be read.
 *
 * @internal
 */
export const publicKeyOf = (family, crypto, privateKey) => Effect.suspend(() => {
  const stored = KeyModel.material(privateKey);
  if (stored._tag === "CryptoKey" && !stored.cryptoKey.extractable) {
    return internal.notExtractable(family.module, "publicKeyOf");
  }
  const spec = family.specOf(privateKey.algorithm);
  return Effect.map(internal.run(family.module, "publicKeyOf", privateKey.algorithm.name, crypto, async subtle => publicOf(subtle, stored._tag === "CryptoKey" ? stored.cryptoKey : await importPrivate(subtle, stored.bytes, spec, true), spec)), bytes => KeyModel.makeBytes(family.publicKeyProto, privateKey.algorithm, bytes));
});
/**
 * Deno imports raw EC points that are not on the curve and rejects them with
 * a `TypeError` when the key is used. For a byte-backed public key only the
 * key can cause that, so it is `InvalidKey`. Any other `TypeError`, such as a
 * `CryptoKey` from another `Crypto` object, stays a defect.
 *
 * @internal
 */
export const publicKeyError = (module, method, publicKey, cause) => KeyModel.material(publicKey)._tag === "Bytes" && Predicate.hasProperty(cause, "name") && cause.name === "TypeError" ? new CryptoError.CryptoError({
  module,
  method,
  reason: new CryptoError.InvalidKey({
    description: "Invalid public key",
    cause
  })
}) : internal.webCryptoError(module, method, publicKey.algorithm.name, cause);
//# sourceMappingURL=asymmetric.js.map