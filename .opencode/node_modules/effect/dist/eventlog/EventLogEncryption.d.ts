import * as Context from "../Context.ts";
import * as Aead from "../crypto/Aead.ts";
import * as Digest from "../crypto/Digest.ts";
import * as SecureRandom from "../crypto/SecureRandom.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
import * as Schema from "../Schema.ts";
import * as Transferable from "../workers/Transferable.ts";
import { type Entry, RemoteEntry } from "./EventJournal.ts";
import type { Identity } from "./EventLog.ts";
/**
 * Schema for a batch of journal entries encrypted as one AES-GCM message.
 *
 * **Details**
 *
 * `entryIds` lists the ids of the encrypted entries in order. It is sent in
 * plaintext so the server can deduplicate batches, and it is authenticated
 * together with the store id, so a server cannot alter it or move the batch to
 * another store.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export declare const EncryptedBatch: Schema.Struct<{
    readonly entryIds: Schema.NonEmptyArray<Schema.brand<Schema.instanceOf<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>, "effect/eventlog/EventJournal/EntryId">>;
    readonly iv: Transferable.Transferable<Schema.instanceOf<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>>;
    readonly encryptedBatch: Transferable.Transferable<Schema.instanceOf<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>>;
}>;
/**
 * Type of an encrypted batch.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface EncryptedBatch extends Schema.Schema.Type<typeof EncryptedBatch> {
}
/**
 * Type of an encrypted batch stored by a remote server, including the remote
 * sequence of its first entry.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface EncryptedRemoteBatch extends Schema.Schema.Type<typeof EncryptedRemoteBatch> {
}
/**
 * Schema for encrypted batches exchanged with a remote event-log server.
 *
 * **Details**
 *
 * The entries of a batch have consecutive remote sequences starting at
 * `sequence`.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export declare const EncryptedRemoteBatch: Schema.Struct<{
    readonly entryIds: Schema.NonEmptyArray<Schema.brand<Schema.instanceOf<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>, "effect/eventlog/EventJournal/EntryId">>;
    readonly iv: Transferable.Transferable<Schema.instanceOf<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>>;
    readonly encryptedBatch: Transferable.Transferable<Schema.instanceOf<Uint8Array<ArrayBuffer>, Uint8Array<ArrayBuffer>>>;
    readonly sequence: Schema.Natural;
}>;
declare const EventLogEncryption_base: Context.ServiceClass<EventLogEncryption, "effect/eventlog/EventLogEncryption", {
    readonly encrypt: (identity: Identity["Service"], storeId: string, entries: ReadonlyArray<Entry>) => Effect.Effect<Array<EncryptedBatch>>;
    readonly decrypt: (identity: Identity["Service"], storeId: string, batches: ReadonlyArray<EncryptedRemoteBatch>) => Effect.Effect<Array<RemoteEntry>>;
    readonly sha256: (data: Uint8Array) => Effect.Effect<Uint8Array>;
    readonly generateIdentity: Effect.Effect<Identity["Service"]>;
}>;
/**
 * Service that provides identity generation, entry
 * encryption and decryption, and SHA-256 hashing for event-log replication.
 *
 * **When to use**
 *
 * Use to provide cryptographic operations required by encrypted event-log
 * replication.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare class EventLogEncryption extends EventLogEncryption_base {
}
/**
 * Creates an `EventLogEncryption` service from the `effect/crypto` services.
 *
 * **Details**
 *
 * Each identity's AES-256-GCM key is derived from its root secret with
 * `Digest` and cached per identity. Batches are encrypted with `Aead` under a
 * random 12-byte nonce from `SecureRandom`, which also generates identities.
 * A batch that fails authentication, or whose entries do not match its entry
 * ids, is a defect.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const make: Effect.Effect<EventLogEncryption["Service"], never, Aead.Aead | Digest.Digest | SecureRandom.SecureRandom>;
/**
 * Provides `EventLogEncryption` from the `Aead`, `Digest` and `SecureRandom`
 * services.
 *
 * **When to use**
 *
 * Use to choose the crypto layers yourself, such as the `NodeCrypto` layers.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layer: Layer.Layer<EventLogEncryption, never, Aead.Aead | Digest.Digest | SecureRandom.SecureRandom>;
/**
 * Creates an `EventLogEncryption` service on the Web Crypto layers of
 * `Aead`, `Digest` and `SecureRandom`, using the supplied `Crypto` object.
 *
 * @stability unstable
 * @category encryption
 * @since 4.0.0
 */
export declare const makeEncryptionSubtle: (crypto: Crypto) => Effect.Effect<EventLogEncryption["Service"]>;
/**
 * Provides `EventLogEncryption` on the Web Crypto layers of `Aead`, `Digest`
 * and `SecureRandom`.
 *
 * **Details**
 *
 * The layers use the `Crypto` object of the `WebCrypto` reference, which
 * defaults to `globalThis.crypto`.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerSubtle: Layer.Layer<EventLogEncryption>;
export {};
//# sourceMappingURL=EventLogEncryption.d.ts.map