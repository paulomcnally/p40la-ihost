/**
 * ChaCha20-Poly1305 (RFC 8439) and XChaCha20-Poly1305
 * (draft-irtf-cfrg-xchacha-03), vendored from `@noble/ciphers`.
 *
 * Upstream: https://github.com/paulmillr/noble-ciphers
 * Version: 2.4.0, commit d9e8a6a599e7ed729d9be03854c46a3c73bd9a79
 * Files: `src/chacha.ts` (`chachaCore`, `hchacha`, `chacha20`, `xchacha20`,
 * `_poly1305_aead`), `src/_arx.ts` (`createCipher`), `src/_poly1305.ts`
 * (`Poly1305`), and the helpers they use from `src/utils.ts`. `@noble/ciphers`
 * 1.0.0 was audited by Cure53 in 2024.
 *
 * Changes from upstream: merged into one file, removed the `TArg`/`TRet` type
 * helpers, `@example` blocks, the reduced-round and original ChaCha variants,
 * the PRGs, and the `wrapCipher` argument wrapper, whose checks the `Aead`
 * layers run before calling this code. `Poly1305` is constructed directly
 * instead of through `wrapMacConstructor`, and `decrypt` returns `undefined`
 * on a tag mismatch instead of throwing, so any exception is a bug. The
 * cipher core, Poly1305 arithmetic, and AEAD construction are unchanged.
 *
 * The MIT License (MIT)
 *
 * Copyright (c) 2022 Paul Miller (https://paulmillr.com)
 * Copyright (c) 2016 Thomas Pornin <pornin@bolet.org>
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 * @internal
 */
export {};
//# sourceMappingURL=chacha.d.ts.map