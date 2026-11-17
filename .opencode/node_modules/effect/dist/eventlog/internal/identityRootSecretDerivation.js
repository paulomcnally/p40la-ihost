import * as Aead from "../../crypto/Aead.js";
import * as Digest from "../../crypto/Digest.js";
import * as KeyFormat from "../../crypto/KeyFormat.js";
import * as SecureRandom from "../../crypto/SecureRandom.js";
import * as Signature from "../../crypto/Signature.js";
import * as WebCrypto from "../../crypto/WebCrypto.js";
import * as Effect from "../../Effect.js";
import * as Layer from "../../Layer.js";
import * as Redacted from "../../Redacted.js";
const textEncoder = /*#__PURE__*/new TextEncoder();
/**
 * @internal
 */
export const EncryptionDerivationLabelV1 = "effect/eventlog/identity/v1/encryption";
/**
 * @internal
 */
export const SigningDerivationLabelV1 = "effect/eventlog/identity/v1/signing";
// The crypto services return fresh arrays over an `ArrayBuffer`.
const toArrayBufferBytes = bytes => bytes;
// Each secret is SHA-256(label || 0x00 || root secret).
const deriveSecretBytes = (rootSecret, label) => Digest.Digest.use(digest => {
  const labelBytes = textEncoder.encode(label);
  const derivationInput = new Uint8Array(labelBytes.byteLength + 1 + rootSecret.byteLength);
  derivationInput.set(labelBytes, 0);
  derivationInput.set(rootSecret, labelBytes.byteLength + 1);
  return digest.digest(Digest.sha256, derivationInput);
});
/**
 * Derives the AES-256-GCM key that encrypts an identity's entries.
 *
 * @internal
 */
export const deriveEncryptionKey = /*#__PURE__*/Effect.fnUntraced(function* (rootSecret) {
  const encryptionKeyMaterial = yield* deriveSecretBytes(rootSecret, EncryptionDerivationLabelV1);
  const encryptionKey = yield* Effect.orDie(Aead.key(Aead.aes256Gcm, encryptionKeyMaterial));
  return {
    encryptionKeyMaterial: toArrayBufferBytes(encryptionKeyMaterial),
    encryptionKey
  };
});
/**
 * Derives the Ed25519 key an identity signs session authentication payloads
 * with.
 *
 * @internal
 */
const deriveSigningKey = /*#__PURE__*/Effect.fnUntraced(function* (rootSecret) {
  const signature = yield* Signature.Signature;
  const seed = yield* deriveSecretBytes(rootSecret, SigningDerivationLabelV1);
  const privateKey = yield* Signature.privateKey(Signature.ed25519, seed);
  const signingPublicKey = yield* Effect.flatMap(signature.publicKeyOf(privateKey), Signature.toBytes);
  const signingPrivateKey = yield* KeyFormat.encode(KeyFormat.pkcs8, privateKey);
  return {
    signingPublicKey: toArrayBufferBytes(signingPublicKey),
    signingPrivateKey: Redacted.make(toArrayBufferBytes(signingPrivateKey))
  };
}, Effect.orDie);
/**
 * @internal
 */
export const deriveIdentityRootSecretMaterial = rootSecret => Effect.zipWith(deriveEncryptionKey(rootSecret), deriveSigningKey(rootSecret), (encryption, signing) => ({
  ...encryption,
  ...signing
}));
/**
 * Memoizes a derivation from an identity's root secret per identity instance.
 *
 * @internal
 */
export const cachedByIdentity = derive => {
  const cache = new WeakMap();
  return identity => Effect.suspend(() => {
    const cached = cache.get(identity);
    if (cached !== undefined) {
      return Effect.succeed(cached);
    }
    return Effect.tap(derive(Redacted.value(identity.privateKey)), derived => Effect.sync(() => {
      cache.set(identity, derived);
    }));
  });
};
const layerWebCrypto = /*#__PURE__*/Layer.mergeAll(Digest.layerWebCrypto, Signature.layerWebCrypto).pipe(/*#__PURE__*/Layer.provide(SecureRandom.layerWebCrypto));
/**
 * Derives identity material on the Web Crypto layers, using `crypto` when it
 * is given and the `WebCrypto` reference otherwise.
 *
 * @internal
 */
export const makeGetIdentityRootSecretMaterial = crypto => cachedByIdentity(rootSecret => crypto === undefined ? Effect.provide(deriveIdentityRootSecretMaterial(rootSecret), layerWebCrypto)
// `local` keeps enclosing Web Crypto layers, built on another `Crypto`,
// from being reused.
: Effect.provide(deriveIdentityRootSecretMaterial(rootSecret), Layer.provideMerge(layerWebCrypto, WebCrypto.layer(crypto)), {
  local: true
}));
//# sourceMappingURL=identityRootSecretDerivation.js.map