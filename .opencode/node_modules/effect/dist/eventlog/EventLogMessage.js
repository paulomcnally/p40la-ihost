import * as Effect from "../Effect.js";
import * as SchemaBinary from "../encoding/SchemaBinary.js";
import * as Predicate from "../Predicate.js";
import * as Rpc from "../rpc/Rpc.js";
import * as RpcGroup from "../rpc/RpcGroup.js";
import * as RpcMiddleware from "../rpc/RpcMiddleware.js";
import * as Schema from "../Schema.js";
import * as Transferable from "../workers/Transferable.js";
import { Entry, EntryFields, entryFromFields, EntryId, RemoteEntry, RemoteId } from "./EventJournal.js";
import { EncryptedBatch, EncryptedRemoteBatch } from "./EventLogEncryption.js";
/**
 * Runtime brand identifier for event-log store ids.
 *
 * @stability unstable
 * @category type IDs
 * @since 4.0.0
 */
export const StoreIdTypeId = "effect/eventlog/EventLog/StoreId";
/**
 * Schema for branded event-log store ids.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const StoreId = /*#__PURE__*/Schema.String.pipe(/*#__PURE__*/Schema.brand(StoreIdTypeId));
/**
 * Error returned by event-log remote RPCs.
 *
 * **Details**
 *
 * It records the request tag, optional identity and store information, a protocol
 * error code, and a human-readable message.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class EventLogProtocolError extends /*#__PURE__*/Schema.TaggedError("effect/eventlog/EventLogRemote/ProtocolError")("EventLogProtocolError", {
  requestTag: Schema.String,
  publicKey: /*#__PURE__*/Schema.optional(Schema.String),
  storeId: /*#__PURE__*/Schema.optional(StoreId),
  code: /*#__PURE__*/Schema.Literals(["Unauthorized", "Forbidden", "NotFound", "InvalidRequest", "InternalServerError"]),
  message: Schema.String
}) {
  /**
   * Returns `true` when the value is an `EventLogProtocolError`.
   *
   * @stability unstable
   * @since 4.0.0
   */
  static is(u) {
    return Predicate.isTagged(u, "EventLogProtocolError");
  }
}
/**
 * RPC middleware that authenticates event-log requests and provides the client
 * `Identity` to authenticated handlers.
 *
 * @stability unstable
 * @category middleware
 * @since 4.0.0
 */
export class EventLogAuthentication extends /*#__PURE__*/RpcMiddleware.Service()("effect/eventlog/EventLogMessage/EventLogAuthentication", {
  error: EventLogProtocolError
}) {}
/**
 * Response sent by the remote server during the authentication handshake.
 *
 * **Details**
 *
 * Contains the remote id, a challenge for the client to sign, and the lowest
 * assigned sequence (default 0) for initializing acknowledgement cursors.
 * `compaction` is `true` when the server serves `EventLogCompactionRpcs`.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class HelloResponse extends /*#__PURE__*/Schema.Class("effect/eventlog/EventLogRemote/HelloResponse")({
  remoteId: RemoteId,
  challenge: Transferable.Uint8Array,
  firstSequence: /*#__PURE__*/Schema.optional(Schema.Natural),
  compaction: /*#__PURE__*/Schema.optional(/*#__PURE__*/Schema.Literal(true))
}) {}
/**
 * RPC used to start an event-log remote session and receive a `HelloResponse`.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class HelloRpc extends /*#__PURE__*/Rpc.make("EventLog.Hello", {
  success: HelloResponse
}) {}
/**
 * Schema for an authentication request containing the client public key,
 * Ed25519 signing public key, signature over the session challenge payload, and
 * algorithm name.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class Authenticate extends /*#__PURE__*/Schema.Class("effect/eventlog/EventLogRemote/Authenticate")({
  publicKey: Schema.String,
  signingPublicKey: Transferable.Uint8Array,
  signature: Transferable.Uint8Array,
  algorithm: /*#__PURE__*/Schema.Literal("Ed25519")
}) {}
/**
 * RPC used to authenticate a remote event-log session after `HelloRpc`.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class AuthenticateRpc extends /*#__PURE__*/Rpc.make("EventLog.Authenticate", {
  payload: Authenticate,
  error: EventLogProtocolError
}) {}
/**
 * Represents an entire encoded event-log payload in one transport frame.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class SingleMessage extends /*#__PURE__*/Schema.TaggedClass("effect/eventlog/EventLogRemote/SingleMessage")("Single", {
  data: Transferable.Uint8Array
}) {}
/**
 * Represents one part of a large encoded event-log payload.
 *
 * **When to use**
 *
 * Use to divide data into chunks and `join` to reassemble all chunks with
 * the same id once every part has arrived.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class ChunkedMessage extends /*#__PURE__*/Schema.TaggedClass("effect/eventlog/EventLogRemote/ChunkedMessage")("Chunked", {
  id: Schema.Int,
  part: /*#__PURE__*/Schema.Tuple([Schema.Natural, Schema.Natural]).check(/*#__PURE__*/Schema.makeFilter(([index, total]) => index < total, {
    expected: "a chunk part with an index less than its total"
  })),
  data: Transferable.Uint8Array
}) {
  static chunkSize = 512_000;
  static initialJoinState() {
    return new Map();
  }
  /**
   * Splits binary event-log message data into numbered chunks.
   *
   * @stability unstable
   * @since 4.0.0
   */
  static split(id, data) {
    const parts = Math.max(1, Math.ceil(data.byteLength / ChunkedMessage.chunkSize));
    const result = new Array(parts);
    for (let i = 0; i < parts; i++) {
      const start = i * ChunkedMessage.chunkSize;
      const end = Math.min((i + 1) * ChunkedMessage.chunkSize, data.byteLength);
      result[i] = new ChunkedMessage({
        id,
        part: [i, parts],
        data: data.subarray(start, end)
      });
    }
    return result;
  }
  /**
   * Reassembles all chunks for a message id into the original binary payload.
   *
   * @stability unstable
   * @since 4.0.0
   */
  static join(map, part) {
    const [index, total] = part.part;
    let entry = map.get(part.id);
    if (!entry) {
      entry = {
        parts: new Array(total),
        count: 0,
        bytes: 0
      };
      map.set(part.id, entry);
    }
    if (entry.parts[index] !== undefined) {
      return;
    }
    entry.parts[index] = part.data;
    entry.count++;
    entry.bytes += part.data.byteLength;
    if (entry.count !== total) {
      return;
    }
    const data = new Uint8Array(entry.bytes);
    let offset = 0;
    for (const part of entry.parts) {
      data.set(part, offset);
      offset += part.byteLength;
    }
    map.delete(part.id);
    return data;
  }
}
/**
 * Schema for a written entry's id and a sequence at which the server stores
 * it. Entries the server already held keep their stored sequence. When a write
 * stores an entry again, as part of a partly known batch, it is acknowledged at
 * its new position.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.6
 */
export const WriteAck = /*#__PURE__*/Schema.Struct({
  entryId: EntryId,
  sequence: Schema.Natural
});
/**
 * RPC used to send one chunk of a large encoded write payload.
 *
 * **Details**
 *
 * The completing chunk returns acknowledgements in request order; earlier
 * chunks return an empty array.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class WriteChunkedRpc extends /*#__PURE__*/Rpc.make("EventLog.WriteChunked", {
  payload: ChunkedMessage,
  success: Schema.Array(WriteAck),
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {}
/**
 * Schema for encrypted event-log write payloads sent to a remote store.
 *
 * **Details**
 *
 * It includes the client public key, target store id, and the entries
 * encrypted as batches.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class WriteEntries extends /*#__PURE__*/Schema.Class("effect/eventlog/EventLogRemote/WriteEntries")({
  publicKey: Schema.String,
  storeId: StoreId,
  batches: /*#__PURE__*/Schema.Array(EncryptedBatch)
}) {
  static FromSchemaBinary = /*#__PURE__*/SchemaBinary.toCodec(WriteEntries);
  static encode = /*#__PURE__*/Schema.encodeEffect(this.FromSchemaBinary);
  static decode = /*#__PURE__*/Schema.decodeEffect(this.FromSchemaBinary);
  get encoded() {
    return WriteEntries.encode(this);
  }
}
/**
 * Schema for plaintext event-log write payloads sent to a remote store.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class WriteEntriesUnencrypted extends /*#__PURE__*/Schema.Class("effect/eventlog/EventLogRemote/WriteEntriesUnencrypted")({
  publicKey: Schema.String,
  storeId: StoreId,
  entries: /*#__PURE__*/Schema.Array(Entry)
}) {
  static FromSchemaBinary = /*#__PURE__*/SchemaBinary.toCodec(WriteEntriesUnencrypted);
  static encode = /*#__PURE__*/Schema.encodeEffect(this.FromSchemaBinary);
  static decode = (data, options) => Effect.map(decodeWriteEntriesUnencryptedFields(data, options), fields => Schema.makeClassTrusted(WriteEntriesUnencrypted, {
    ...fields,
    entries: fields.entries.map(entryFromFields)
  }));
  get encoded() {
    return WriteEntriesUnencrypted.encode(this);
  }
}
const decodeWriteEntriesUnencryptedFields = /*#__PURE__*/Schema.decodeEffect(/*#__PURE__*/SchemaBinary.toCodec(/*#__PURE__*/Schema.Struct({
  ...WriteEntriesUnencrypted.fields,
  entries: /*#__PURE__*/Schema.Array(EntryFields)
})));
/**
 * RPC used to send an encoded write payload that fits in one message.
 *
 * **Details**
 *
 * The response acknowledges every written entry, in request order.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class WriteSingleRpc extends /*#__PURE__*/Rpc.make("EventLog.WriteSingle", {
  payload: {
    data: Transferable.Uint8Array
  },
  success: Schema.Array(WriteAck),
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {}
/**
 * RPC used to stream remote event-log changes for a public key and store id
 * starting at a sequence number.
 *
 * **Details**
 *
 * Responses are encoded as either `SingleMessage` values or `ChunkedMessage`
 * parts. Entries written on this connection are acknowledged, not echoed.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class ChangesRpc extends /*#__PURE__*/Rpc.make("EventLog.Changes", {
  payload: {
    publicKey: Schema.String,
    storeId: StoreId,
    startSequence: Schema.Natural
  },
  success: Schema.Union([SingleMessage, ChunkedMessage]),
  error: EventLogProtocolError,
  stream: true
}).middleware(EventLogAuthentication) {
  static EncryptedFromSchemaBinary = /*#__PURE__*/SchemaBinary.toCodec(/*#__PURE__*/Schema.NonEmptyArray(EncryptedRemoteBatch));
  static UnencryptedFromSchemaBinary = /*#__PURE__*/SchemaBinary.toCodec(/*#__PURE__*/Schema.NonEmptyArray(RemoteEntry));
  static encodeEncrypted = /*#__PURE__*/Schema.encodeEffect(ChangesRpc.EncryptedFromSchemaBinary);
  static decodeEncrypted = /*#__PURE__*/Schema.decodeEffect(ChangesRpc.EncryptedFromSchemaBinary);
  static encodeUnencrypted = /*#__PURE__*/Schema.encodeEffect(ChangesRpc.UnencryptedFromSchemaBinary);
  static decodeUnencrypted = /*#__PURE__*/Schema.decodeEffect(ChangesRpc.UnencryptedFromSchemaBinary);
}
/**
 * RPC group containing the event-log remote handshake, authentication, write, and
 * changes endpoints.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class EventLogRemoteRpcs extends /*#__PURE__*/RpcGroup.make(HelloRpc, AuthenticateRpc, WriteChunkedRpc, WriteSingleRpc, ChangesRpc) {}
const compactionScope = {
  publicKey: Schema.String,
  storeId: StoreId
};
const compactionStage = {
  ...compactionScope,
  stageId: Schema.String
};
/**
 * The compaction state of a store, as reported to clients.
 *
 * **Details**
 *
 * `wanted` is `true` when the server would open a compaction stage. `head` is
 * the highest stored sequence.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionStatus extends /*#__PURE__*/Schema.Class("effect/eventlog/EventLogRemote/CompactionStatus")({
  wanted: Schema.Boolean,
  head: Schema.Int
}) {}
/**
 * RPC that reports a store's compaction state.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionStatusRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionStatus", {
  payload: compactionScope,
  success: CompactionStatus,
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {}
/**
 * Schema for the result of opening a compaction stage.
 *
 * **Details**
 *
 * `Started` pins the stage to the active generation and to the last input
 * sequence. `Busy` means another stage is open on the store. `NotWanted` means
 * the store doesn't need compacting.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export const CompactionBeginResult = /*#__PURE__*/Schema.Union([/*#__PURE__*/Schema.TaggedStruct("Started", {
  stageId: Schema.String
}), /*#__PURE__*/Schema.TaggedStruct("Busy", {}), /*#__PURE__*/Schema.TaggedStruct("NotWanted", {})]);
/**
 * RPC that opens a compaction stage.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionBeginRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionBegin", {
  payload: compactionScope,
  success: CompactionBeginResult,
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {}
/**
 * Schema for the header of one compaction input batch.
 *
 * **Details**
 *
 * An entry's input sequence is the batch `sequence` plus its position in
 * `entryIds`.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export const CompactionInputHeader = /*#__PURE__*/Schema.Struct({
  sequence: Schema.Natural,
  entryIds: /*#__PURE__*/Schema.NonEmptyArray(EntryId)
});
/**
 * RPC that streams a stage's input batch headers in input order.
 *
 * **Details**
 *
 * The input is the pinned snapshot, then the originals up to the stage's
 * `through`. Headers are framed as `SingleMessage` values or `ChunkedMessage`
 * parts, as in `ChangesRpc`. It fails with a `NotFound` error once the stage
 * or its pin is gone.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionInputRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionInput", {
  payload: compactionStage,
  success: Schema.Union([SingleMessage, ChunkedMessage]),
  error: EventLogProtocolError,
  stream: true
}).middleware(EventLogAuthentication) {
  static HeadersFromSchemaBinary = /*#__PURE__*/SchemaBinary.toCodec(/*#__PURE__*/Schema.NonEmptyArray(CompactionInputHeader));
  static encodeHeaders = /*#__PURE__*/Schema.encodeEffect(CompactionInputRpc.HeadersFromSchemaBinary);
  static decodeHeaders = /*#__PURE__*/Schema.decodeEffect(CompactionInputRpc.HeadersFromSchemaBinary);
}
/**
 * RPC that streams the encrypted input batches at the given input sequences.
 *
 * **Details**
 *
 * `batches` holds the input sequence of each batch to fetch. Batches are framed
 * and encoded as in `ChangesRpc`. It fails with a `NotFound` error once the
 * stage or its pin is gone.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionFetchRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionFetch", {
  payload: {
    ...compactionStage,
    batches: Schema.Array(Schema.Natural)
  },
  success: Schema.Union([SingleMessage, ChunkedMessage]),
  error: EventLogProtocolError,
  stream: true
}).middleware(EventLogAuthentication) {}
/**
 * RPC that uploads one piece of a compaction snapshot segment.
 *
 * **Details**
 *
 * A segment is an encoded non-empty array of encrypted batches, indexed by
 * `chunk.id`, and is split into pieces when larger than
 * `ChunkedMessage.chunkSize`. Re-uploading a stored segment is acknowledged
 * without storing it again. A segment that fails validation aborts the stage
 * with an `InvalidRequest` error.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionPartRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionPart", {
  payload: {
    ...compactionStage,
    chunk: ChunkedMessage
  },
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {
  static SegmentFromSchemaBinary = /*#__PURE__*/SchemaBinary.toCodec(/*#__PURE__*/Schema.NonEmptyArray(EncryptedBatch));
  static encodeSegment = /*#__PURE__*/Schema.encodeEffect(CompactionPartRpc.SegmentFromSchemaBinary);
  static decodeSegment = /*#__PURE__*/Schema.decodeEffect(CompactionPartRpc.SegmentFromSchemaBinary);
}
/**
 * RPC that releases an open compaction stage. A late abort is a no-op.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionAbortRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionAbort", {
  payload: compactionStage,
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {}
/**
 * Schema for the outcome of a compaction run reported at commit.
 *
 * **Details**
 *
 * `Snapshot` activates the `segmentCount` uploaded segments. `NoOp` reports a
 * run that dropped nothing.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export const CompactionResult = /*#__PURE__*/Schema.Union([/*#__PURE__*/Schema.TaggedStruct("Snapshot", {
  segmentCount: Schema.Natural
}), /*#__PURE__*/Schema.TaggedStruct("NoOp", {})]);
/**
 * RPC that commits a compaction stage.
 *
 * **Details**
 *
 * Answers `Superseded` when the stage's pinned generation is no longer active.
 * Fails with `InvalidRequest`, aborting the stage, when the segments are
 * incomplete or the kept ids are not a non-empty, repeat-free subsequence of
 * the pinned input. Fails with `NotFound` when the stage expired, was aborted
 * or committed, or is unknown, so a retried commit fails too.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class CompactionCommitRpc extends /*#__PURE__*/Rpc.make("EventLog.CompactionCommit", {
  payload: {
    ...compactionStage,
    result: CompactionResult
  },
  success: Schema.Literals(["Committed", "Superseded"]),
  error: EventLogProtocolError
}).middleware(EventLogAuthentication) {}
/**
 * RPC group for encrypted remote compaction.
 *
 * **Details**
 *
 * Served by the encrypted server alongside `EventLogRemoteRpcs`. Clients call
 * it only when `HelloResponse.compaction` is set.
 *
 * @stability unstable
 * @category protocols
 * @since 4.0.0
 */
export class EventLogCompactionRpcs extends /*#__PURE__*/RpcGroup.make(CompactionStatusRpc, CompactionBeginRpc, CompactionInputRpc, CompactionFetchRpc, CompactionPartRpc, CompactionAbortRpc, CompactionCommitRpc) {}
//# sourceMappingURL=EventLogMessage.js.map