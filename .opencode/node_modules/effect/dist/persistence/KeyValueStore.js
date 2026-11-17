/**
 * Provides effectful key/value storage for persistence backends.
 *
 * `KeyValueStore` is a service for storing string or binary values by key. It
 * is useful for lightweight durable state, browser storage, local files, SQL
 * tables, tests, and as a storage building block for higher-level persistence
 * APIs. This module includes store operations, prefixed views, schema-aware JSON
 * storage, error values, and layers for memory, filesystem, Web Storage, and
 * SQL-backed stores.
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Context from "../Context.js";
import * as Data from "../Data.js";
import * as Effect from "../Effect.js";
import * as Base64 from "../encoding/Base64.js";
import * as FileSystem from "../FileSystem.js";
import { dual, identity } from "../Function.js";
import * as Layer from "../Layer.js";
import * as Option from "../Option.js";
import * as Path from "../Path.js";
import * as Predicate from "../Predicate.js";
import * as Result from "../Result.js";
import * as Schema from "../Schema.js";
import * as SqlClient from "../sql/SqlClient.js";
import * as UndefinedOr from "../UndefinedOr.js";
const TypeId = "~effect/persistence/KeyValueStore";
const ErrorTypeId = "~effect/persistence/KeyValueStore/KeyValueStoreError";
/**
 * Error raised by key/value store operations, including the failed method,
 * optional key, message, and cause.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class KeyValueStoreError extends /*#__PURE__*/Data.TaggedError("KeyValueStoreError") {
  /**
   * Marks this value as a key-value store error for runtime guards.
   *
   * @stability unstable
   * @since 4.0.0
   */
  [ErrorTypeId] = ErrorTypeId;
}
/**
 * Service tag for string and binary key/value storage.
 *
 * **When to use**
 *
 * Use to access or provide the persistence store used for lightweight durable
 * state.
 *
 * @stability unstable
 * @category services
 * @since 4.0.0
 */
export const KeyValueStore = /*#__PURE__*/Context.Service("effect/persistence/KeyValueStore");
/**
 * Constructs a `KeyValueStore` from primitive store operations.
 *
 * **Details**
 *
 * Default implementations are derived for `has`, `isEmpty`, `clearPrefix`,
 * `modify`, and `modifyUint8Array` unless they are provided in the options.
 *
 * If `clearPrefix` is not provided, it delegates to `clear` for an empty prefix
 * and otherwise removes each key returned by `keys` that starts with the
 * prefix.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const make = options => KeyValueStore.of({
  [TypeId]: TypeId,
  has: key => Effect.map(options.get(key), Predicate.isNotUndefined),
  isEmpty: Effect.map(options.size, size => size === 0),
  clearPrefix: prefix => prefix.length === 0 ? options.clear : Effect.flatMap(options.keys, keys => Effect.forEach(keys, key => key.startsWith(prefix) ? options.remove(key) : Effect.void, {
    discard: true
  })),
  modify: (key, f) => Effect.flatMap(options.get(key), o => {
    if (o === undefined) {
      return Effect.undefined;
    }
    const newValue = f(o);
    return Effect.as(options.set(key, newValue), newValue);
  }),
  modifyUint8Array: (key, f) => Effect.flatMap(options.getUint8Array(key), o => {
    if (o === undefined) {
      return Effect.undefined;
    }
    const newValue = f(o);
    return Effect.as(options.set(key, newValue), newValue);
  }),
  ...options
});
/**
 * Adapts a string-only backing store into a `KeyValueStore`.
 *
 * **Details**
 *
 * `Uint8Array` values are stored as base64 strings. `getUint8Array` decodes
 * base64 values and falls back to UTF-8 encoding for non-base64 strings.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const makeStringOnly = options => {
  const encoder = new TextEncoder();
  return make({
    ...options,
    getUint8Array: key => options.get(key).pipe(Effect.map(UndefinedOr.map(value => Result.match(Base64.decode(value), {
      onFailure: () => encoder.encode(value),
      onSuccess: identity
    })))),
    set: (key, value) => typeof value === "string" ? options.set(key, value) : Effect.suspend(() => options.set(key, Base64.encode(value)))
  });
};
/**
 * Returns a view of a `KeyValueStore` that prepends the given prefix to every
 * key.
 *
 * **Details**
 *
 * `keys`, `size`, `isEmpty`, and `clear` only see entries whose key starts
 * with the prefix, and `keys` returns them with the prefix removed. `keys`,
 * `size`, and `isEmpty` list every key in the underlying store and filter the
 * result, while `clear` delegates to `clearPrefix`.
 *
 * @stability unstable
 * @category combinators
 * @since 4.0.0
 */
export const prefix = /*#__PURE__*/dual(2, (self, prefix) => {
  const keys = Effect.map(self.keys, keys => keys.flatMap(key => key.startsWith(prefix) ? [key.slice(prefix.length)] : []));
  return {
    ...self,
    get: key => self.get(`${prefix}${key}`),
    getUint8Array: key => self.getUint8Array(`${prefix}${key}`),
    set: (key, value) => self.set(`${prefix}${key}`, value),
    remove: key => self.remove(`${prefix}${key}`),
    clear: self.clearPrefix(prefix),
    clearPrefix: key => self.clearPrefix(`${prefix}${key}`),
    keys,
    size: Effect.map(keys, keys => keys.length),
    isEmpty: Effect.map(keys, keys => keys.length === 0),
    has: key => self.has(`${prefix}${key}`),
    modify: (key, f) => self.modify(`${prefix}${key}`, f),
    modifyUint8Array: (key, f) => self.modifyUint8Array(`${prefix}${key}`, f)
  };
});
/**
 * Provides a process-local in-memory `KeyValueStore` backed by a `Map`.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layerMemory = /*#__PURE__*/Layer.sync(KeyValueStore)(() => {
  const store = new Map();
  const encoder = new TextEncoder();
  return make({
    get: key => Effect.sync(() => {
      const value = store.get(key);
      return value === undefined ? undefined : typeof value === "string" ? value : Base64.encode(value);
    }),
    getUint8Array: key => Effect.sync(() => {
      const value = store.get(key);
      return value === undefined ? undefined : typeof value === "string" ? encoder.encode(value) : value;
    }),
    set: (key, value) => Effect.sync(() => store.set(key, value)),
    remove: key => Effect.sync(() => store.delete(key)),
    clear: Effect.sync(() => store.clear()),
    clearPrefix: prefix => Effect.sync(() => {
      for (const key of store.keys()) {
        if (key.startsWith(prefix)) store.delete(key);
      }
    }),
    size: Effect.sync(() => store.size),
    keys: Effect.sync(() => Array.from(store.keys()))
  });
});
/**
 * Provides a `KeyValueStore` backed by files in the specified directory.
 *
 * **Details**
 *
 * The directory is created if needed, and each key is percent-encoded as a
 * single file name. Empty keys, `.` and `..` are rejected. Keys are only
 * guaranteed to be distinct on case-sensitive file systems.
 *
 * `clear` removes the directory recursively, so it must not be shared with
 * unrelated data.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layerFileSystem = directory => Layer.effect(KeyValueStore)(Effect.gen(function* () {
  const fs = yield* FileSystem.FileSystem;
  const path = yield* Path.Path;
  const withKeyPath = (method, key, f) => key.length === 0 || key === "." || key === ".." ? Effect.fail(new KeyValueStoreError({
    method,
    key,
    message: `Invalid key ${key}`
  })) : f(path.join(directory, encodeURIComponent(key)));
  if (!(yield* fs.exists(directory))) {
    yield* fs.makeDirectory(directory, {
      recursive: true
    });
  }
  return make({
    get: key => withKeyPath("get", key, path => Effect.catchTag(fs.readFileString(path), "PlatformError", cause => cause.reason._tag === "NotFound" ? Effect.undefined : Effect.fail(new KeyValueStoreError({
      method: "get",
      key,
      message: `Unable to get item with key ${key}`,
      cause
    })))),
    getUint8Array: key => withKeyPath("getUint8Array", key, path => Effect.catchTag(fs.readFile(path), "PlatformError", cause => cause.reason._tag === "NotFound" ? Effect.undefined : Effect.fail(new KeyValueStoreError({
      method: "getUint8Array",
      key,
      message: `Unable to get item with key ${key}`,
      cause
    })))),
    set: (key, value) => withKeyPath("set", key, path => Effect.mapError(typeof value === "string" ? fs.writeFileString(path, value) : fs.writeFile(path, value), cause => new KeyValueStoreError({
      method: "set",
      key,
      message: `Unable to set item with key ${key}`,
      cause
    }))),
    remove: key => withKeyPath("remove", key, path => Effect.mapError(fs.remove(path, {
      force: true
    }), cause => new KeyValueStoreError({
      method: "remove",
      key,
      message: `Unable to remove item with key ${key}`,
      cause
    }))),
    has: key => withKeyPath("has", key, path => Effect.mapError(fs.exists(path), cause => new KeyValueStoreError({
      method: "has",
      key,
      message: `Unable to check existence of item with key ${key}`,
      cause
    }))),
    clear: Effect.mapError(Effect.andThen(fs.remove(directory, {
      recursive: true
    }), fs.makeDirectory(directory, {
      recursive: true
    })), cause => new KeyValueStoreError({
      method: "clear",
      message: `Unable to clear storage`,
      cause
    })),
    clearPrefix: prefix => fs.readDirectory(directory).pipe(Effect.flatMap(files => Effect.forEach(files.filter(file => decodeFileKey(file)?.startsWith(prefix)), file => Effect.catchTag(fs.remove(path.join(directory, file)), "PlatformError", cause => cause.reason._tag === "NotFound" ? Effect.void : Effect.fail(cause)), {
      discard: true
    })), Effect.mapError(cause => new KeyValueStoreError({
      method: "clearPrefix",
      key: prefix,
      message: `Unable to clear items with prefix ${prefix}`,
      cause
    }))),
    size: Effect.matchEffect(fs.readDirectory(directory), {
      onSuccess: files => Effect.succeed(files.length),
      onFailure: cause => Effect.fail(new KeyValueStoreError({
        method: "size",
        message: `Unable to get size`,
        cause
      }))
    }),
    keys: fs.readDirectory(directory).pipe(Effect.flatMap(files => Effect.try(() => files.map(decodeURIComponent))), Effect.mapError(cause => new KeyValueStoreError({
      method: "keys",
      message: `Unable to get keys`,
      cause
    })))
  });
}));
const decodeFileKey = file => {
  try {
    return decodeURIComponent(file);
  } catch {
    return undefined;
  }
};
const escapeLike = prefix => `${prefix.replace(/[!%_[\]]/g, "!$&")}%`;
/**
 * Provides a SQL-backed `KeyValueStore`.
 *
 * **Details**
 *
 * The layer creates the configured table if it does not exist and stores both
 * string and binary values through the current `SqlClient`.
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layerSql = (options = {}) => Layer.effect(KeyValueStore)(Effect.gen(function* () {
  const sql = (yield* SqlClient.SqlClient).withoutTransforms();
  const tableName = options.table ?? "effect_key_value_store";
  const table = sql(tableName);
  yield* sql.onDialectOrElse({
    mysql: () => sql`
          CREATE TABLE IF NOT EXISTS ${table} (
            id VARCHAR(191) PRIMARY KEY,
            value BLOB NOT NULL,
            value_type SMALLINT NOT NULL
          )
        `,
    pg: () => sql`
          CREATE TABLE IF NOT EXISTS ${table} (
            id TEXT PRIMARY KEY,
            value BYTEA NOT NULL,
            value_type SMALLINT NOT NULL
          )
        `,
    mssql: () => sql`
          IF NOT EXISTS (SELECT * FROM sysobjects WHERE name=${tableName} AND xtype='U')
          CREATE TABLE ${table} (
            id NVARCHAR(450) PRIMARY KEY,
            value VARBINARY(MAX) NOT NULL,
            value_type SMALLINT NOT NULL
          )
        `,
    // sqlite
    orElse: () => sql`
          CREATE TABLE IF NOT EXISTS ${table} (
            id TEXT PRIMARY KEY,
            value BLOB NOT NULL,
            value_type INTEGER NOT NULL
          )
        `
  }).pipe(Effect.orDie);
  const upsert = sql.onDialectOrElse({
    pg: () => entry => sql`
          INSERT INTO ${table} (id, value, value_type) VALUES (${entry.id}, ${entry.value}, ${entry.value_type})
          ON CONFLICT (id) DO UPDATE SET value=EXCLUDED.value, value_type=EXCLUDED.value_type
        `.unprepared,
    mysql: () => entry => sql`
          INSERT INTO ${table} (id, value, value_type) VALUES (${entry.id}, ${entry.value}, ${entry.value_type})
          ON DUPLICATE KEY UPDATE value=VALUES(value), value_type=VALUES(value_type)
        `,
    mssql: () => entry => sql`
          MERGE ${table} AS target
          USING (SELECT ${entry.id} AS id, ${entry.value} AS value, ${entry.value_type} AS value_type) AS source
          ON target.id = source.id
          WHEN MATCHED THEN UPDATE SET value = source.value, value_type = source.value_type
          WHEN NOT MATCHED THEN INSERT (id, value, value_type)
          VALUES (source.id, source.value, source.value_type);
        `,
    // sqlite
    orElse: () => entry => sql`
          INSERT INTO ${table} (id, value, value_type) VALUES (${entry.id}, ${entry.value}, ${entry.value_type})
          ON CONFLICT(id) DO UPDATE SET value=excluded.value, value_type=excluded.value_type
        `.unprepared
  });
  // The first LIKE uses the primary key index. On mysql and mssql it follows
  // the column collation, which is usually case / accent-insensitive, so a
  // second binary LIKE keeps only exact prefix matches.
  const deleteByPrefix = sql.onDialectOrElse({
    // sqlite LIKE is case-insensitive, so compare the leading characters
    orElse: () => prefix => sql`DELETE FROM ${table} WHERE substr(id, 1, length(${prefix})) = ${prefix}`,
    pg: () => prefix => sql`DELETE FROM ${table} WHERE id LIKE ${escapeLike(prefix)} ESCAPE '!'`,
    mysql: () => prefix => {
      const pattern = escapeLike(prefix);
      return sql`DELETE FROM ${table} WHERE id LIKE ${pattern} ESCAPE '!' AND CONVERT(id USING utf8mb4) COLLATE utf8mb4_bin LIKE ${pattern} ESCAPE '!'`;
    },
    mssql: () => prefix => {
      const pattern = escapeLike(prefix);
      return sql`DELETE FROM ${table} WHERE id LIKE ${pattern} ESCAPE '!' AND id COLLATE Latin1_General_100_BIN2 LIKE ${pattern} ESCAPE '!'`;
    }
  });
  const encoder = new TextEncoder();
  const decoder = new TextDecoder();
  const ValueTypeString = 0;
  const ValueTypeUint8Array = 1;
  return make({
    get: key => sql`SELECT value, value_type FROM ${table} WHERE id = ${key}`.pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "get",
      key,
      message: `Unable to get item with key ${key}`,
      cause
    })), Effect.flatMap(rows => {
      if (rows.length === 0) {
        return Effect.undefined;
      }
      const row = rows[0];
      switch (row.value_type) {
        case ValueTypeString:
          return Effect.succeed(decoder.decode(row.value));
        case ValueTypeUint8Array:
          return Effect.succeed(Base64.encode(row.value));
        default:
          return Effect.fail(new KeyValueStoreError({
            method: "get",
            key,
            message: `Invalid stored value type for key ${key}: ${row.value_type}`
          }));
      }
    })),
    getUint8Array: key => sql`SELECT value, value_type FROM ${table} WHERE id = ${key}`.pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "getUint8Array",
      key,
      message: `Unable to get item with key ${key}`,
      cause
    })), Effect.flatMap(rows => {
      if (rows.length === 0) {
        return Effect.undefined;
      }
      const row = rows[0];
      switch (row.value_type) {
        case ValueTypeString:
          return Effect.succeed(row.value);
        case ValueTypeUint8Array:
          return Effect.succeed(row.value);
        default:
          return Effect.fail(new KeyValueStoreError({
            method: "getUint8Array",
            key,
            message: `Invalid stored value type for key ${key}: ${row.value_type}`
          }));
      }
    })),
    set: (key, value) => upsert({
      id: key,
      value: typeof value === "string" ? encoder.encode(value) : value,
      value_type: typeof value === "string" ? ValueTypeString : ValueTypeUint8Array
    }).pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "set",
      key,
      message: `Unable to set item with key ${key}`,
      cause
    })), Effect.asVoid),
    remove: key => sql`DELETE FROM ${table} WHERE id = ${key}`.pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "remove",
      key,
      message: `Unable to remove item with key ${key}`,
      cause
    })), Effect.asVoid),
    clear: sql`DELETE FROM ${table}`.pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "clear",
      message: `Unable to clear storage`,
      cause
    })), Effect.asVoid),
    clearPrefix: prefix => deleteByPrefix(prefix).pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "clearPrefix",
      key: prefix,
      message: `Unable to clear items with prefix ${prefix}`,
      cause
    })), Effect.asVoid),
    size: sql`SELECT COUNT(*) as count FROM ${table}`.pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "size",
      message: `Unable to get size`,
      cause
    })), Effect.map(rows => rows.length === 0 ? 0 : Number(rows[0].count))),
    keys: sql`SELECT id FROM ${table}`.pipe(Effect.mapError(cause => new KeyValueStoreError({
      method: "keys",
      message: `Unable to get keys`,
      cause
    })), Effect.map(rows => rows.map(row => row.id)))
  });
}));
const SchemaStoreTypeId = "~effect/persistence/KeyValueStore/SchemaStore";
/**
 * Adapts a `KeyValueStore` into a `SchemaStore` using the schema's JSON codec.
 *
 * @stability unstable
 * @category converting
 * @since 4.0.0
 */
export const toSchemaStore = (self, schema) => {
  const serializer = Schema.toCodecJson(schema);
  const jsonSchema = Schema.fromJsonString(serializer);
  const decode = Schema.decodeEffect(jsonSchema);
  const encode = Schema.encodeEffect(jsonSchema);
  const get = key => Effect.flatMap(self.get(key), UndefinedOr.match({
    onUndefined: () => Effect.succeedNone,
    onDefined: value => Effect.asSome(decode(value))
  }));
  const set = (key, value) => Effect.flatMap(encode(value), json => self.set(key, json));
  const modify = (key, f) => Effect.flatMap(get(key), o => {
    if (Option.isNone(o)) {
      return Effect.succeedNone;
    }
    const newValue = f(o.value);
    return Effect.as(set(key, newValue), Option.some(newValue));
  });
  return {
    [SchemaStoreTypeId]: SchemaStoreTypeId,
    get,
    set,
    modify,
    remove: self.remove,
    clear: self.clear,
    size: self.size,
    keys: self.keys,
    has: self.has,
    isEmpty: self.isEmpty
  };
};
/**
 * Provides a `KeyValueStore` backed by a Web `Storage` instance such as
 * `localStorage` or `sessionStorage`.
 *
 * **Details**
 *
 * This layer uses the Web Storage API:
 * https://developer.mozilla.org/en-US/docs/Web/API/Web_Storage_API
 *
 * @stability unstable
 * @category layers
 * @since 4.0.0
 */
export const layerStorage = evaluate => Layer.sync(KeyValueStore)(() => {
  const storage = evaluate();
  return makeStringOnly({
    get: key => Effect.try({
      try: () => storage.getItem(key) ?? undefined,
      catch: cause => new KeyValueStoreError({
        key,
        method: "get",
        message: `Unable to get item with key ${key}`,
        cause
      })
    }),
    set: (key, value) => Effect.try({
      try: () => storage.setItem(key, value),
      catch: cause => new KeyValueStoreError({
        key,
        method: "set",
        message: `Unable to set item with key ${key}`,
        cause
      })
    }),
    remove: key => Effect.try({
      try: () => storage.removeItem(key),
      catch: cause => new KeyValueStoreError({
        key,
        method: "remove",
        message: `Unable to remove item with key ${key}`,
        cause
      })
    }),
    clear: Effect.try({
      try: () => storage.clear(),
      catch: cause => new KeyValueStoreError({
        method: "clear",
        message: `Unable to clear storage`,
        cause
      })
    }),
    clearPrefix: prefix => Effect.try({
      try: () => {
        const keys = [];
        for (let i = 0; i < storage.length; i++) {
          const key = storage.key(i);
          if (key !== null && key.startsWith(prefix)) keys.push(key);
        }
        for (const key of keys) storage.removeItem(key);
      },
      catch: cause => new KeyValueStoreError({
        key: prefix,
        method: "clearPrefix",
        message: `Unable to clear items with prefix ${prefix}`,
        cause
      })
    }),
    size: Effect.try({
      try: () => storage.length,
      catch: cause => new KeyValueStoreError({
        method: "size",
        message: `Unable to get size`,
        cause
      })
    }),
    keys: Effect.try({
      try: () => {
        const keys = [];
        for (let i = 0; i < storage.length; i++) {
          const key = storage.key(i);
          if (key !== null) keys.push(key);
        }
        return keys;
      },
      catch: cause => new KeyValueStoreError({
        method: "keys",
        message: `Unable to get keys`,
        cause
      })
    })
  });
});
//# sourceMappingURL=KeyValueStore.js.map