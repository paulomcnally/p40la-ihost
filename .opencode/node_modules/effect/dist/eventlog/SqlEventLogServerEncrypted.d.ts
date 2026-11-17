import * as Digest from "../crypto/Digest.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
import type * as Scope from "../Scope.ts";
import * as SqlClient from "../sql/SqlClient.ts";
import * as SqlError from "../sql/SqlError.ts";
import * as EventLogServerEncrypted from "./EventLogServerEncrypted.ts";
/**
 * Creates encrypted event-log server `Storage` backed by SQL.
 *
 * **Details**
 *
 * It persists the server remote id, session authentication bindings, and encrypted
 * batches in dialect-specific tables, creating per-identity/store batch and entry
 * id tables as needed.
 *
 * The storage always provides `compaction`. Its tables are created on startup
 * and when a store is first opened. A sweep in the surrounding scope prunes
 * superseded generations every `sweepInterval`. Startup dies on invalid
 * compaction options (see `EventLogServerEncrypted.makeCompactionConfig`).
 *
 * **Gotchas**
 *
 * Storage does not retry sequence conflicts between competing instances.
 * `EventLog.runRemote` retries failed writes; direct `EventLogRemote.write`
 * callers must retry failures other than Forbidden themselves. Live notifications
 * are local to each storage instance; clients on other instances must reconnect
 * to read those writes.
 *
 * Inside a caller's transaction, `write` and `compaction.rollback` publish to
 * live readers after the outermost commit, and not at all if it rolls back.
 * Concurrent caller transactions may reach live readers out of sequence order.
 * `changes` always reads in committed order.
 *
 * Instances sharing a database should share compaction options. The instance
 * that writes a generation or stage fixes its prune time or expiry.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeStorage: (options?: {
    readonly entryTablePrefix?: string;
    readonly remoteIdTable?: string;
    readonly compaction?: EventLogServerEncrypted.CompactionOptions | undefined;
}) => Effect.Effect<EventLogServerEncrypted.Storage["Service"], SqlError.SqlError, SqlClient.SqlClient | Digest.Digest | Scope.Scope>;
/**
 * Provides encrypted server `Storage` using the SQL-backed implementation.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerStorage: (options?: {
    readonly entryTablePrefix?: string;
    readonly remoteIdTable?: string;
    readonly compaction?: EventLogServerEncrypted.CompactionOptions | undefined;
}) => Layer.Layer<EventLogServerEncrypted.Storage, SqlError.SqlError, SqlClient.SqlClient | Digest.Digest>;
/**
 * Provides SQL-backed encrypted server `Storage` and supplies the default Web
 * Crypto `Digest` layer.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerStorageSubtle: (options?: {
    readonly entryTablePrefix?: string;
    readonly remoteIdTable?: string;
    readonly compaction?: EventLogServerEncrypted.CompactionOptions | undefined;
}) => Layer.Layer<EventLogServerEncrypted.Storage, SqlError.SqlError, SqlClient.SqlClient>;
//# sourceMappingURL=SqlEventLogServerEncrypted.d.ts.map