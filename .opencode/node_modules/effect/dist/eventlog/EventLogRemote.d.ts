import * as Context from "../Context.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
import * as Queue from "../Queue.ts";
import * as RpcClient from "../rpc/RpcClient.ts";
import type { RpcClientError } from "../rpc/RpcClientError.ts";
import type * as Schema from "../Schema.ts";
import type * as Scope from "../Scope.ts";
import * as Stream from "../Stream.ts";
import { type Entry, type EventJournal, type EventJournalError, RemoteEntry, type RemoteId } from "./EventJournal.ts";
import { type Identity, Registry } from "./EventLog.ts";
import { EventLogEncryption } from "./EventLogEncryption.ts";
import { ChunkedMessage, type CompactionBeginResult, type CompactionInputHeader, type CompactionResult, type CompactionStatus, EventLogProtocolError, type HelloResponse, type SingleMessage, type StoreId, type WriteAck } from "./EventLogMessage.ts";
declare const EventLogRemote_base: Context.ServiceClass<EventLogRemote, "effect/eventlog/EventLogRemote", {
    readonly id: RemoteId;
    /**
     * The lowest sequence the remote assigns, or 0 when omitted.
     */
    readonly firstSequence?: number | undefined;
    readonly changes: (options: {
        readonly identity: Identity["Service"];
        readonly storeId: StoreId;
        readonly startSequence: number;
        /**
         * Looks up locally stored entries by id, so the remote can reuse them
         * instead of decoding the matching changes.
         */
        readonly knownEntries?: EventJournal["Service"]["entriesById"] | undefined;
    }) => Effect.Effect<Queue.Dequeue<RemoteEntry, EventLogRemoteError>, never, Scope.Scope>;
    readonly write: (options: {
        readonly identity: Identity["Service"];
        readonly storeId: StoreId;
        readonly entries: ReadonlyArray<Entry>;
    }) => Effect.Effect<ReadonlyArray<WriteAck> | void, EventLogRemoteError>;
    readonly whenAuthenticated: <A, E, R>(effect: Effect.Effect<A, E, R>) => Effect.Effect<A, E | EventLogRemoteError, R | Identity>;
    /**
     * Encrypted remote compaction. Present when the remote advertises
     * `HelloResponse.compaction` and the RPC client includes
     * `EventLogCompactionRpcs`.
     */
    readonly compaction?: EventLogRemoteCompaction | undefined;
}>;
/**
 * Service that represents a remote event-log replica.
 *
 * **When to use**
 *
 * Use to access or provide a remote event-log replica that can write local
 * entries and stream remote changes.
 *
 * **Details**
 *
 * It can write local entries to the remote, stream remote changes from a sequence
 * number, and run effects only after the supplied identity has authenticated.
 *
 * Built-in remotes acknowledge writes instead of echoing them through `changes`.
 * Custom remotes may return `void` from `write` to confirm uploads through
 * `changes` instead.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare class EventLogRemote extends EventLogRemote_base {
}
/**
 * A stage opened by `EventLogRemoteCompaction.begin`, with the identity and
 * store it belongs to.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface CompactionStage {
    readonly identity: Identity["Service"];
    readonly storeId: StoreId;
    readonly stageId: string;
}
/**
 * Client side of encrypted remote compaction.
 *
 * **Details**
 *
 * A run opens a stage with `begin`, reads its input headers with `input`,
 * fetches the batches it needs with `fetch`, uploads the kept entries with
 * `upload` and calls `commit`. `abort` releases a stage that won't be
 * committed.
 *
 * @stability unstable
 * @category models
 * @since 4.0.0
 */
export interface EventLogRemoteCompaction {
    readonly status: (options: {
        readonly identity: Identity["Service"];
        readonly storeId: StoreId;
    }) => Effect.Effect<CompactionStatus, EventLogRemoteError>;
    readonly begin: (options: {
        readonly identity: Identity["Service"];
        readonly storeId: StoreId;
    }) => Effect.Effect<CompactionBeginResult, EventLogRemoteError>;
    /**
     * Streams the stage's input batch headers in input order.
     */
    readonly input: (stage: CompactionStage) => Stream.Stream<CompactionInputHeader, EventLogRemoteError>;
    /**
     * Fetches and decrypts the input batches at the given input sequences.
     */
    readonly fetch: (stage: CompactionStage, sequences: ReadonlyArray<number>) => Stream.Stream<RemoteEntry, EventLogRemoteError>;
    /**
     * Encrypts the kept entries, in input-sequence order, and uploads them in
     * segments of about `ChunkedMessage.chunkSize`. An oversized segment is
     * split into pieces, and a segment is resent in full after a dropped
     * connection.
     */
    readonly upload: (stage: CompactionStage, entries: ReadonlyArray<Entry>) => Effect.Effect<{
        readonly segmentCount: number;
    }, EventLogRemoteError>;
    readonly commit: (stage: CompactionStage, result: CompactionResult) => Effect.Effect<"Committed" | "Superseded", EventLogRemoteError>;
    /**
     * Releases the stage. Failures are ignored.
     */
    readonly abort: (stage: CompactionStage) => Effect.Effect<void>;
}
declare const EventLogRemoteError_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => import("../Cause.ts").YieldableError & {
    readonly _tag: "EventLogRemoteError";
} & Readonly<A>;
/**
 * Error raised by `EventLogRemote` operations, recording the failed method and
 * underlying cause.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export declare class EventLogRemoteError extends EventLogRemoteError_base<{
    readonly method: string;
    readonly cause: unknown;
}> {
    /**
     * Returns `true` when the value is an `EventLogRemoteError`.
     *
     * @stability unstable
     * @since 4.0.0
     */
    static is(u: unknown): u is EventLogRemoteError;
}
declare const EventLogRemoteClient_base: Context.ServiceClass<EventLogRemoteClient, "effect/eventlog/EventLogRemote/EventLogRemoteClient", {
    readonly "EventLog.Authenticate": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly signingPublicKey: Uint8Array<ArrayBuffer>;
        readonly signature: Uint8Array<ArrayBuffer>;
        readonly algorithm: "Ed25519";
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : void, RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
    readonly "EventLog.Changes": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
        readonly startSequence: number;
    }, options?: {
        readonly asQueue?: AsQueue | undefined;
        readonly streamBufferSize?: number | undefined;
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
    } | undefined) => AsQueue extends true ? Effect.Effect<Queue.Dequeue<ChunkedMessage | SingleMessage, import("../Cause.ts").Done<void> | EventLogProtocolError | RpcClientError>, never, Scope.Scope> : Stream.Stream<ChunkedMessage | SingleMessage, EventLogProtocolError | RpcClientError, never>;
    readonly "EventLog.Hello": <const AsQueue extends boolean = false, const Discard = false>(input: void, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : HelloResponse, RpcClientError | (Discard extends true ? never : never), never>;
    readonly "EventLog.WriteChunked": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly _tag?: "Chunked";
        readonly id: number;
        readonly part: readonly [number, number];
        readonly data: Uint8Array<ArrayBuffer>;
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : readonly {
        readonly entryId: Uint8Array<ArrayBuffer> & import("../Brand.ts").Brand<"effect/eventlog/EventJournal/EntryId">;
        readonly sequence: number;
    }[], EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
    readonly "EventLog.WriteSingle": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly data: Uint8Array<ArrayBuffer>;
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : readonly {
        readonly entryId: Uint8Array<ArrayBuffer> & import("../Brand.ts").Brand<"effect/eventlog/EventJournal/EntryId">;
        readonly sequence: number;
    }[], EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
} & Partial<{
    readonly "EventLog.CompactionAbort": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
        readonly stageId: string;
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : void, EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
    readonly "EventLog.CompactionBegin": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : {
        readonly _tag: "Started";
        readonly stageId: string;
    } | {
        readonly _tag: "Busy";
    } | {
        readonly _tag: "NotWanted";
    }, EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
    readonly "EventLog.CompactionCommit": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
        readonly stageId: string;
        readonly result: {
            readonly _tag?: "Snapshot";
            readonly segmentCount: number;
        } | {
            readonly _tag?: "NoOp";
        };
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : "Committed" | "Superseded", EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
    readonly "EventLog.CompactionFetch": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
        readonly stageId: string;
        readonly batches: readonly number[];
    }, options?: {
        readonly asQueue?: AsQueue | undefined;
        readonly streamBufferSize?: number | undefined;
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
    } | undefined) => AsQueue extends true ? Effect.Effect<Queue.Dequeue<ChunkedMessage | SingleMessage, import("../Cause.ts").Done<void> | EventLogProtocolError | RpcClientError>, never, Scope.Scope> : Stream.Stream<ChunkedMessage | SingleMessage, EventLogProtocolError | RpcClientError, never>;
    readonly "EventLog.CompactionInput": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
        readonly stageId: string;
    }, options?: {
        readonly asQueue?: AsQueue | undefined;
        readonly streamBufferSize?: number | undefined;
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
    } | undefined) => AsQueue extends true ? Effect.Effect<Queue.Dequeue<ChunkedMessage | SingleMessage, import("../Cause.ts").Done<void> | EventLogProtocolError | RpcClientError>, never, Scope.Scope> : Stream.Stream<ChunkedMessage | SingleMessage, EventLogProtocolError | RpcClientError, never>;
    readonly "EventLog.CompactionPart": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
        readonly stageId: string;
        readonly chunk: ChunkedMessage;
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : void, EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
    readonly "EventLog.CompactionStatus": <const AsQueue extends boolean = false, const Discard = false>(input: {
        readonly publicKey: string;
        readonly storeId: string & import("../Brand.ts").Brand<"effect/eventlog/EventLog/StoreId">;
    }, options?: {
        readonly headers?: import("../http/Headers.ts").Input | undefined;
        readonly context?: Context.Context<never> | undefined;
        readonly discard?: Discard | undefined;
    } | undefined) => Effect.Effect<Discard extends true ? void : CompactionStatus, EventLogProtocolError | RpcClientError | (Discard extends true ? never : EventLogProtocolError), never>;
}>>;
/**
 * Service that provides a typed RPC client for the `EventLogRemoteRpcs` protocol.
 *
 * **When to use**
 *
 * Use to provide the RPC client used by remote event-log replicas to
 * authenticate, write entries, and subscribe to changes.
 *
 * **Details**
 *
 * The `EventLogCompactionRpcs` methods are optional. The encrypted remote
 * offers compaction only when the client has them. `layer` builds the client
 * from both groups.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export declare class EventLogRemoteClient extends EventLogRemoteClient_base {
    static readonly layer: Layer.Layer<EventLogRemoteClient, never, RpcClient.Protocol>;
}
/**
 * Creates an `EventLogRemote` from custom write encoding and change decoding
 * functions.
 *
 * **Details**
 *
 * The remote performs the hello/authentication handshake, retries after forbidden
 * responses by re-authenticating, chunks large writes, and registers itself with
 * the `Registry` for the current scope.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeWith: (options: {
    readonly encodeWrite: (options: {
        readonly identity: Identity["Service"];
        readonly entries: ReadonlyArray<Entry>;
        readonly storeId: StoreId;
    }) => Effect.Effect<Uint8Array<ArrayBuffer>, Schema.SchemaError>;
    readonly decodeChanges: (identity: Identity["Service"], storeId: StoreId, data: Uint8Array<ArrayBuffer>, knownEntries?: EventJournal["Service"]["entriesById"]) => Effect.Effect<ReadonlyArray<RemoteEntry>, Schema.SchemaError | EventJournalError>;
}) => Effect.Effect<EventLogRemote["Service"], EventLogRemoteError, Scope.Scope | EventLogRemoteClient | Registry>;
/**
 * Creates an `EventLogRemote` that encrypts outgoing entries and decrypts
 * incoming changes with `EventLogEncryption`.
 *
 * **Details**
 *
 * The remote has a `compaction` member when the server advertises compaction
 * and the RPC client includes `EventLogCompactionRpcs`, unless built with
 * `compaction: false`. Without it the client never calls the compaction RPCs.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeEncryptedWith: (options: {
    readonly compaction?: boolean | undefined;
}) => Effect.Effect<{
    readonly id: RemoteId;
    /**
     * The lowest sequence the remote assigns, or 0 when omitted.
     */
    readonly firstSequence?: number | undefined;
    readonly changes: (options: {
        readonly identity: Identity["Service"];
        readonly storeId: StoreId;
        readonly startSequence: number;
        /**
         * Looks up locally stored entries by id, so the remote can reuse them
         * instead of decoding the matching changes.
         */
        readonly knownEntries?: EventJournal["Service"]["entriesById"] | undefined;
    }) => Effect.Effect<Queue.Dequeue<RemoteEntry, EventLogRemoteError>, never, Scope.Scope>;
    readonly write: (options: {
        readonly identity: Identity["Service"];
        readonly storeId: StoreId;
        readonly entries: ReadonlyArray<Entry>;
    }) => Effect.Effect<ReadonlyArray<WriteAck> | void, EventLogRemoteError>;
    readonly whenAuthenticated: <A, E, R>(effect: Effect.Effect<A, E, R>) => Effect.Effect<A, E | EventLogRemoteError, R | Identity>;
    /**
     * Encrypted remote compaction. Present when the remote advertises
     * `HelloResponse.compaction` and the RPC client includes
     * `EventLogCompactionRpcs`.
     */
    readonly compaction?: EventLogRemoteCompaction | undefined;
}, EventLogRemoteError, EventLogEncryption | EventLogRemoteClient | Registry | Scope.Scope>;
/**
 * Creates an `EventLogRemote` that encrypts outgoing entries and decrypts
 * incoming changes with `EventLogEncryption`.
 *
 * @see {@link makeEncryptedWith} for opting out of running compaction
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeEncrypted: Effect.Effect<EventLogRemote["Service"], EventLogRemoteError, Scope.Scope | EventLogRemoteClient | EventLogEncryption | Registry>;
/**
 * Creates an `EventLogRemote` that sends and receives plaintext entry payloads.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeUnencrypted: Effect.Effect<EventLogRemote["Service"], EventLogRemoteError, Scope.Scope | EventLogRemoteClient | Registry>;
/**
 * Provides an encrypted `EventLogRemote` using the remote RPC client and the
 * default Web Crypto encryption layer.
 *
 * **Details**
 *
 * `compaction: false` leaves the remote without its `compaction` member, so
 * this client never calls the compaction RPCs.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerEncryptedWith: (options: {
    readonly compaction?: boolean | undefined;
}) => Layer.Layer<EventLogRemote, EventLogRemoteError, RpcClient.Protocol | Registry>;
/**
 * Provides an encrypted `EventLogRemote` using the remote RPC client and the
 * default Web Crypto encryption layer.
 *
 * @see {@link layerEncryptedWith} for opting out of running compaction
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerEncrypted: Layer.Layer<EventLogRemote, EventLogRemoteError, RpcClient.Protocol | Registry>;
/**
 * Provides an unencrypted `EventLogRemote` using the remote RPC client.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerUnencrypted: Layer.Layer<EventLogRemote, EventLogRemoteError, RpcClient.Protocol | Registry>;
export {};
//# sourceMappingURL=EventLogRemote.d.ts.map