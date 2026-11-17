import * as Digest from "../../crypto/Digest.js";
import { webCryptoDigest } from "../../crypto/internal/digest.js";
import { WebCrypto } from "../../crypto/WebCrypto.js";
import * as Effect from "../../Effect.js";
import * as Hex from "../../encoding/Hex.js";
const encoder = /*#__PURE__*/new TextEncoder();
/**
 * Execution ids and idempotency keys are persisted, so they keep the first 16
 * bytes of the SHA-256 digest as 32 hex characters.
 *
 * @internal
 */
export const makeHashDigest = original => WebCrypto.use(crypto => Effect.map(webCryptoDigest(crypto, Digest.sha256, encoder.encode(original)), bytes => Hex.encode(bytes).slice(0, 32)));
//# sourceMappingURL=crypto.js.map