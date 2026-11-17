import * as Digest from "../crypto/Digest.ts";
import * as Effect from "../Effect.ts";
import * as Layer from "../Layer.ts";
import * as Migrator from "../sql/Migrator.ts";
import * as SqlClient from "../sql/SqlClient.ts";
import * as MessageStorage from "./MessageStorage.ts";
import type { ShardingConfig } from "./ShardingConfig.ts";
import * as Snowflake from "./Snowflake.ts";
/**
 * Creates a SQL-backed encoded message storage driver, running its migrations
 * and using the optional table prefix.
 *
 * **When to use**
 *
 * Use when you need the SQL-backed encoded driver directly, such as when
 * composing a custom message storage adapter.
 *
 * **Details**
 *
 * The optional `prefix` controls the table names for messages, replies, and
 * migrations; when omitted, `cluster` is used.
 *
 * **Gotchas**
 *
 * Changing `prefix` after deployment points the runtime at a different set of
 * tables, including the migration history table.
 *
 * @see {@link make} for the decoded `MessageStorage` constructor
 * @see {@link makeEncodedStorage} for construction without migrations
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const makeEncoded: (options?: {
    readonly prefix?: string | undefined;
}) => Effect.Effect<MessageStorage.Encoded, never, SqlClient.SqlClient | Digest.Digest>;
/**
 * Creates a SQL-backed encoded message storage driver without running migrations
 * or executing startup SQL.
 *
 * **When to use**
 *
 * Use when composing a custom encoded adapter with a runtime SQL connection
 * that does not have schema-management privileges.
 *
 * **Details**
 *
 * The optional `prefix` controls the message and reply table names; when omitted,
 * `cluster` is used.
 *
 * **Gotchas**
 *
 * Run `layerMigrations` separately with the same prefix before using the driver.
 * This constructor does not create tables or check that the schema is up to date.
 *
 * @see {@link makeEncoded} for construction that runs migrations
 * @see {@link layerMigrations} for running migrations separately
 * @see {@link layerStorage} for providing decoded storage without migrations
 *
 * @stability unstable
 * @category constructors
 * @since 4.1.0
 */
export declare const makeEncodedStorage: (options?: {
    readonly prefix?: string | undefined;
}) => Effect.Effect<MessageStorage.Encoded, never, SqlClient.SqlClient | Digest.Digest>;
/**
 * Creates a SQL-backed `MessageStorage` implementation, running its migrations
 * and using the optional table prefix.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export declare const make: (options?: {
    readonly prefix?: string | undefined;
}) => Effect.Effect<MessageStorage.MessageStorage["Service"], never, SqlClient.SqlClient | Snowflake.Generator | Digest.Digest>;
/**
 * Migration loader for the SQL message storage tables.
 *
 * **Details**
 *
 * History is recorded in `<prefix>_migrations`; the default
 * prefix is `cluster`.
 *
 * @stability unstable
 * @category migrations
 * @since 4.1.0
 */
export declare const migrations: (options?: {
    readonly prefix?: string | undefined;
}) => Migrator.Loader;
/**
 * Runs the SQL message storage migrations without providing storage.
 *
 * **Details**
 *
 * Use an owner connection in a deploy step, then use `layerStorage` with a
 * DML-only runtime connection. History is recorded in `<prefix>_migrations`.
 * Migration errors become defects.
 *
 * On PostgreSQL, use the role's `search_path` to select a schema. A
 * schema-qualified prefix such as `app.cluster` produces an invalid index name.
 *
 * @stability unstable
 * @category layers
 * @since 4.1.0
 */
export declare const layerMigrations: (options: {
    readonly prefix?: string | undefined;
}) => Layer.Layer<never, never, SqlClient.SqlClient>;
/**
 * Provides SQL-backed `MessageStorage` without DDL or startup schema checks.
 *
 * **Details**
 *
 * Run `layerMigrations` separately before using this layer.
 * This layer supplies `Snowflake.layerGenerator` internally.
 *
 * @stability unstable
 * @category layers
 * @since 4.1.0
 */
export declare const layerStorage: (options: {
    readonly prefix?: string | undefined;
}) => Layer.Layer<MessageStorage.MessageStorage, never, SqlClient.SqlClient | ShardingConfig | Digest.Digest>;
/**
 * Provides SQL-backed `MessageStorage` with a custom table prefix,
 * running migrations first.
 *
 * **Details**
 *
 * Migration errors become defects. Use `layerMigrations` and `layerStorage`
 * to migrate separately.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layerWith: (options: {
    readonly prefix?: string | undefined;
}) => Layer.Layer<MessageStorage.MessageStorage, never, SqlClient.SqlClient | ShardingConfig | Digest.Digest>;
/**
 * Provides SQL-backed `MessageStorage` with the `cluster` table prefix,
 * running migrations first and supplying `Snowflake.layerGenerator`.
 *
 * **Details**
 *
 * The connection needs DDL permissions. PostgreSQL takes an exclusive lock
 * on the migration history table. Use `layerWith` for a custom prefix or
 * `layerMigrations` and `layerStorage` to migrate separately.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export declare const layer: Layer.Layer<MessageStorage.MessageStorage, never, SqlClient.SqlClient | ShardingConfig | Digest.Digest>;
//# sourceMappingURL=SqlMessageStorage.d.ts.map