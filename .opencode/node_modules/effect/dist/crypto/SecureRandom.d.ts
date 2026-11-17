/**
 * Cryptographically secure randomness: random bytes, uniform integers,
 * shuffles, UUIDv4, UUIDv7, and ULIDs.
 *
 * The {@link SecureRandom} service is built from one `randomBytes` primitive
 * with {@link make}. {@link layerWebCrypto} draws from `getRandomValues` on
 * the `WebCrypto` reference. The `*Unsafe` functions draw synchronously from
 * `globalThis.crypto` for code that cannot use a service, such as constructors
 * and model defaults.
 *
 * Every operation has a `never` error channel. A bad size or an empty range is
 * a defect.
 *
 * **Gotchas**
 *
 * There are no float, boolean, or full-range integer helpers. Insecure random
 * numbers belong to `effect/Random`, and a secure bit is `randomBytes(1)`.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Context from "../Context.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
/**
 * Generates a UUID as a lowercase, hyphenated string or, with
 * `format: "bytes"`, the 16 UUID bytes.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface UuidGenerator {
    (options?: {
        readonly format?: "string" | undefined;
    }): Effect.Effect<string>;
    <Format extends "string" | "bytes">(options: {
        readonly format: Format;
    }): Effect.Effect<Format extends "bytes" ? Uint8Array : string>;
}
declare const SecureRandom_base: Context.ServiceClass<SecureRandom, "effect/crypto/SecureRandom", {
    /**
     * Generates `size` cryptographically secure random bytes.
     *
     * A negative or non-integer size dies with a `RangeError`.
     */
    readonly randomBytes: (size: number) => Effect.Effect<Uint8Array>;
    /**
     * Generates an integer between `min` and `max`, both inclusive unless
     * `halfOpen` excludes `max`.
     *
     * **Details**
     *
     * The result is exactly uniform: draws that would bias it are rejected.
     * `min` is rounded up and `max` down. An empty range, or bounds that do not
     * round to safe integers, die with a `RangeError`.
     */
    readonly intBetween: (min: number, max: number, options?: {
        readonly halfOpen?: boolean | undefined;
    }) => Effect.Effect<number>;
    /**
     * Returns a uniformly shuffled copy of the elements, using Fisher-Yates over
     * `intBetween`.
     */
    readonly shuffle: <A>(elements: Iterable<A>) => Effect.Effect<Array<A>>;
    /**
     * Generates a UUIDv4, as a lowercase hyphenated string or, with
     * `format: "bytes"`, the 16 UUID bytes.
     */
    readonly uuidV4: UuidGenerator;
    /**
     * Generates a UUIDv7 from the `Clock` timestamp, as a lowercase hyphenated
     * string or, with `format: "bytes"`, the 16 UUID bytes.
     *
     * Every UUIDv7 from this service sorts after the previous one, as a string
     * and as bytes. Within a millisecond, a 42-bit counter seeded from random
     * bytes orders them. If the `Clock` moves back, the last timestamp is kept,
     * and if the counter overflows, the timestamp advances by 1 ms, so a
     * timestamp can run slightly ahead of the `Clock`. The ordering ends at the
     * maximum timestamp, 2^48 − 1 ms (in the year 10889), which overflow cannot
     * advance. There is no ordering between separate services, processes or
     * workers.
     *
     * Consecutive UUIDv7s reveal their generation order, so never use them as
     * secrets. Uniqueness between generators is probabilistic: each UUID has 32
     * fresh random bits, on top of a counter seeded from 41 random bits each
     * millisecond.
     */
    readonly uuidV7: UuidGenerator;
    /**
     * Generates a ULID from the `Clock` timestamp: 26 uppercase Crockford
     * base32 characters.
     *
     * Every ULID from this service sorts after the previous one, using the ULID
     * spec's monotonic mode: within a millisecond, the 80-bit random part is
     * incremented. If the `Clock` moves back, the last timestamp is kept, and if
     * the random part overflows, the timestamp advances by 1 ms. The ordering
     * ends at the maximum timestamp, 2^48 − 1 ms (in the year 10889), which
     * overflow cannot advance. There is no ordering between separate services,
     * processes or workers. Consecutive ULIDs reveal their generation order, so
     * never use them as secrets.
     */
    readonly ulid: Effect.Effect<string>;
}>;
/**
 * Service that generates cryptographically secure random values.
 *
 * **Details**
 *
 * Every operation is derived from `randomBytes`. See {@link make}.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare class SecureRandom extends SecureRandom_base {
}
/**
 * Creates a `SecureRandom` service from a secure byte source.
 *
 * **When to use**
 *
 * Use to build the service for a platform integration, a custom runtime, or a
 * seeded test double.
 *
 * **Details**
 *
 * Every operation is derived from `randomBytes`. Sizes are checked before it
 * is called, and an exception it throws is a defect. Each service orders its
 * own UUIDv7s and ULIDs, independently of other services and of the
 * `*Unsafe` functions.
 *
 * **Gotchas**
 *
 * `randomBytes` must return a fresh array of exactly `size` secure random
 * bytes. UUID formatting mutates the returned array.
 *
 * **Example** (Creating a SecureRandom service)
 *
 * ```ts import.meta.vitest
 * import { Effect } from "effect"
 * import { SecureRandom } from "effect/crypto"
 *
 * const random = SecureRandom.make((size) => globalThis.crypto.getRandomValues(new Uint8Array(size)))
 *
 * const bytes = await Effect.runPromise(random.randomBytes(4))
 * bytes.length // => 4
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const make: (randomBytes: (size: number) => Uint8Array) => SecureRandom["Service"];
/**
 * Layer that provides `SecureRandom` from `getRandomValues` on the `WebCrypto`
 * reference.
 *
 * **Details**
 *
 * Requests above the 65,536-byte `getRandomValues` limit are filled in chunks.
 * Only `getRandomValues` is needed, so the layer also works in insecure browser
 * contexts, which have no `crypto.subtle`.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerWebCrypto: Layer.Layer<SecureRandom>;
/**
 * Synchronously generates `size` cryptographically secure random bytes from
 * `globalThis.crypto`.
 *
 * **When to use**
 *
 * Use when a service cannot be read, as in constructors and model defaults.
 * Elsewhere, use the `SecureRandom` service.
 *
 * **Gotchas**
 *
 * Reads `globalThis.crypto` directly, so neither the `WebCrypto` reference nor
 * a mocked `SecureRandom` service affects it. A negative or non-integer size
 * throws a `RangeError`.
 *
 * @stability unstable
 * @category unsafe
 * @since 4.0.0
 */
export declare const randomBytesUnsafe: (size: number) => Uint8Array;
/**
 * Synchronously generates a UUIDv4 from `globalThis.crypto`, as a lowercase
 * hyphenated string or, with `format: "bytes"`, the 16 UUID bytes.
 *
 * **Gotchas**
 *
 * Reads `globalThis.crypto` directly, so neither the `WebCrypto` reference nor
 * a mocked `SecureRandom` service affects it.
 *
 * @stability unstable
 * @category unsafe
 * @since 4.0.0
 */
export declare const uuidV4Unsafe: {
    /**
     * Synchronously generates a UUIDv4 from `globalThis.crypto`, as a lowercase
     * hyphenated string or, with `format: "bytes"`, the 16 UUID bytes.
     *
     * **Gotchas**
     *
     * Reads `globalThis.crypto` directly, so neither the `WebCrypto` reference nor
     * a mocked `SecureRandom` service affects it.
     *
     * @stability unstable
     * @category unsafe
     * @since 4.0.0
     */
    (options?: {
        readonly format?: "string" | undefined;
    }): string;
    /**
     * Synchronously generates a UUIDv4 from `globalThis.crypto`, as a lowercase
     * hyphenated string or, with `format: "bytes"`, the 16 UUID bytes.
     *
     * **Gotchas**
     *
     * Reads `globalThis.crypto` directly, so neither the `WebCrypto` reference nor
     * a mocked `SecureRandom` service affects it.
     *
     * @stability unstable
     * @category unsafe
     * @since 4.0.0
     */
    <Format extends "string" | "bytes">(options: {
        readonly format: Format;
    }): Format extends "bytes" ? Uint8Array : string;
};
/**
 * Synchronously generates a UUIDv7 from `Date.now()` and `globalThis.crypto`,
 * as a lowercase hyphenated string or, with `format: "bytes"`, the 16 UUID
 * bytes.
 *
 * **Details**
 *
 * Below the maximum timestamp, every UUIDv7 generated in this JS realm
 * (process or worker) sorts after the previous one, as a string and as bytes,
 * using the counter described on the `SecureRandom` service's `uuidV7`. The
 * ordering is shared with `Model.UuidV7WithGenerate`, and does not hold across
 * processes or workers. Consecutive UUIDv7s reveal their
 * generation order, so never use them as secrets.
 *
 * **Gotchas**
 *
 * Reads `Date.now()` rather than the `Clock`, and `globalThis.crypto` rather
 * than the `WebCrypto` reference. A timestamp can run slightly ahead of
 * `Date.now()` when the clock moves back or the counter overflows.
 *
 * @stability unstable
 * @category unsafe
 * @since 4.0.0
 */
export declare const uuidV7Unsafe: {
    /**
     * Synchronously generates a UUIDv7 from `Date.now()` and `globalThis.crypto`,
     * as a lowercase hyphenated string or, with `format: "bytes"`, the 16 UUID
     * bytes.
     *
     * **Details**
     *
     * Below the maximum timestamp, every UUIDv7 generated in this JS realm
     * (process or worker) sorts after the previous one, as a string and as bytes,
     * using the counter described on the `SecureRandom` service's `uuidV7`. The
     * ordering is shared with `Model.UuidV7WithGenerate`, and does not hold across
     * processes or workers. Consecutive UUIDv7s reveal their
     * generation order, so never use them as secrets.
     *
     * **Gotchas**
     *
     * Reads `Date.now()` rather than the `Clock`, and `globalThis.crypto` rather
     * than the `WebCrypto` reference. A timestamp can run slightly ahead of
     * `Date.now()` when the clock moves back or the counter overflows.
     *
     * @stability unstable
     * @category unsafe
     * @since 4.0.0
     */
    (options?: {
        readonly format?: "string" | undefined;
    }): string;
    /**
     * Synchronously generates a UUIDv7 from `Date.now()` and `globalThis.crypto`,
     * as a lowercase hyphenated string or, with `format: "bytes"`, the 16 UUID
     * bytes.
     *
     * **Details**
     *
     * Below the maximum timestamp, every UUIDv7 generated in this JS realm
     * (process or worker) sorts after the previous one, as a string and as bytes,
     * using the counter described on the `SecureRandom` service's `uuidV7`. The
     * ordering is shared with `Model.UuidV7WithGenerate`, and does not hold across
     * processes or workers. Consecutive UUIDv7s reveal their
     * generation order, so never use them as secrets.
     *
     * **Gotchas**
     *
     * Reads `Date.now()` rather than the `Clock`, and `globalThis.crypto` rather
     * than the `WebCrypto` reference. A timestamp can run slightly ahead of
     * `Date.now()` when the clock moves back or the counter overflows.
     *
     * @stability unstable
     * @category unsafe
     * @since 4.0.0
     */
    <Format extends "string" | "bytes">(options: {
        readonly format: Format;
    }): Format extends "bytes" ? Uint8Array : string;
};
/**
 * Synchronously generates a ULID from `Date.now()` and `globalThis.crypto`.
 *
 * **Details**
 *
 * Below the maximum timestamp, every ULID generated in this JS realm (process
 * or worker) sorts after the previous one, using the monotonic mode described
 * on the `SecureRandom` service's `ulid`. The ordering does not hold across
 * processes or workers. Consecutive ULIDs reveal their generation order, so
 * never use them as secrets.
 *
 * **Gotchas**
 *
 * Reads `Date.now()` rather than the `Clock`, and `globalThis.crypto` rather
 * than the `WebCrypto` reference. A timestamp can run slightly ahead of
 * `Date.now()` when the clock moves back or the random part overflows.
 *
 * @stability unstable
 * @category unsafe
 * @since 4.0.0
 */
export declare const ulidUnsafe: () => string;
export {};
//# sourceMappingURL=SecureRandom.d.ts.map