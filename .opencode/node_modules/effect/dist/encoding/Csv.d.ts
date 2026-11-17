/**
 * Encodes and decodes CSV streams, with optional schema transformations.
 *
 * **Details**
 *
 * Follows RFC 4180 with a configurable delimiter. Cells are strings, and
 * whitespace and empty cells are kept as they are. Quoted fields can contain
 * delimiters, line breaks and doubled quotes. Decoders accept LF, CRLF and CR
 * line endings and strip a leading BOM. Schema adapters convert cells to and
 * from typed values.
 *
 * @stability unstable
 * @since 4.1.0
 */
import * as Arr from "../Array.ts";
import * as Cause from "../Cause.ts";
import * as Channel from "../Channel.ts";
import type * as Record from "../Record.ts";
import * as Schema from "../Schema.ts";
import * as SchemaAST from "../SchemaAST.ts";
declare const CsvErrorTypeId = "~effect/encoding/Csv/CsvError";
declare const CsvError_base: new <A extends Record<string, any> = {}>(args: import("../Types.ts").VoidIfEmpty<{ readonly [P in keyof A as P extends "_tag" ? never : P]: A[P]; }>) => Cause.YieldableError & {
    readonly _tag: "CsvError";
} & Readonly<A>;
/**
 * A CSV decoding failure.
 *
 * **Details**
 *
 * `record` is one-based and counts the header row and blank lines. `line` is
 * the one-based physical line where the record starts. `column` is the
 * one-based cell index of a syntax error, and `undefined` otherwise.
 *
 * @stability unstable
 * @category errors
 * @since 4.1.0
 */
export declare class CsvError extends CsvError_base<{
    readonly kind: "Syntax" | "RecordSize" | "Header";
    readonly message: string;
    readonly record: number;
    readonly line: number;
    readonly column?: number | undefined;
}> {
    /**
     * Identifies CSV errors across module instances.
     *
     * @stability unstable
     * @since 4.1.0
     */
    readonly [CsvErrorTypeId] = "~effect/encoding/Csv/CsvError";
}
/**
 * Checks whether a value is a `CsvError`.
 *
 * @stability unstable
 * @category guards
 * @since 4.1.0
 */
export declare const isCsvError: (u: unknown) => u is CsvError;
/**
 * A record as an array of string cells.
 *
 * @stability unstable
 * @category models
 * @since 4.1.0
 */
export type Row = ReadonlyArray<string>;
/**
 * CSV decoding options.
 *
 * **Details**
 *
 * - `delimiter`: a single character other than a quote or line break. Defaults
 *   to `","`.
 * - `maxRecordSize`: the maximum record length in UTF-16 code units, including
 *   quotes and delimiters but not the record separator. Defaults to 1 MiB;
 *   `Infinity` disables the limit.
 * - `ignoreEmptyLines`: skips records with no characters. Defaults to `false`,
 *   where an empty line is a record with one empty cell.
 * - `headers`: uses the first record as unique field names and emits the
 *   remaining records as objects of the same width. Defaults to `false`.
 *
 * Invalid options throw when the channel is created.
 *
 * @stability unstable
 * @category configuration
 * @since 4.1.0
 */
export interface DecodeOptions {
    readonly delimiter?: string | undefined;
    readonly maxRecordSize?: number | undefined;
    readonly ignoreEmptyLines?: boolean | undefined;
    readonly headers?: boolean | undefined;
}
/**
 * CSV encoding options.
 *
 * **Details**
 *
 * - `delimiter`: a single character other than a quote or line break. Defaults
 *   to `","`.
 * - `newline`: the record separator written after every record. Defaults to
 *   `"\r\n"`.
 * - `columns`: encodes object records by these keys, in order. Missing or
 *   `undefined` values become empty cells.
 * - `headers`: writes `columns` as the first record, even for an empty stream.
 *   Requires `columns`.
 *
 * Invalid options throw when the channel is created.
 *
 * @stability unstable
 * @category configuration
 * @since 4.1.0
 */
export interface EncodeOptions {
    readonly delimiter?: string | undefined;
    readonly newline?: "\n" | "\r\n" | undefined;
    readonly columns?: ReadonlyArray<string> | undefined;
    readonly headers?: boolean | undefined;
}
/**
 * Schema decoding options.
 *
 * **Details**
 *
 * By default the schema is wrapped in `Schema.toCodecStringTree`, so columns
 * can use schemas such as `Schema.Number`, `Schema.Boolean` or `Schema.Date`.
 * With header records, an empty cell for a property that is optional in the
 * encoded schema is decoded as a missing key. Set `disableCodecs` to decode
 * with the schema exactly as given.
 *
 * @stability unstable
 * @category configuration
 * @since 4.1.0
 */
export interface DecodeSchemaOptions extends DecodeOptions {
    readonly disableCodecs?: boolean | undefined;
}
/**
 * Schema encoding options.
 *
 * **Details**
 *
 * By default the schema is wrapped in `Schema.toCodecStringTree`, so every
 * column encodes to a string; set `disableCodecs` to encode with the schema
 * exactly as given. `columns` defaults to the encoded keys of a struct schema
 * and is required for other object schemas. Tuple and array schemas write
 * cells by position and use `columns` only as header names. Every column must
 * encode to a string, otherwise the channel throws when it is created.
 *
 * @stability unstable
 * @category configuration
 * @since 4.1.0
 */
export type EncodeSchemaOptions<S extends Schema.Constraint> = Omit<EncodeOptions, "columns"> & {
    readonly columns?: (S["Encoded"] extends ReadonlyArray<unknown> ? ReadonlyArray<string> : ReadonlyArray<S["Encoded"] extends infer E ? E extends unknown ? Extract<keyof E, string> : never : never>) | undefined;
    readonly disableCodecs?: boolean | undefined;
};
/**
 * Creates a channel that parses text chunks into CSV records.
 *
 * **Details**
 *
 * Records are arrays of cells, or objects keyed by the first record when
 * `headers` is `true`. Parser state is created per channel run.
 *
 * **Example** (Parsing header records)
 *
 * ```ts import.meta.vitest
 * import { Effect, Stream } from "effect"
 * import { Csv } from "effect/encoding"
 *
 * const records = Stream.make("name,city\nAlice,\"San\nFrancisco\"\n").pipe(
 *   Stream.pipeThroughChannel(Csv.decodeString({ headers: true })),
 *   Stream.runCollect
 * )
 * await Effect.runPromise(records) // => [{ name: "Alice", city: "San\nFrancisco" }]
 * ```
 *
 * @see decode
 * @see decodeSchemaString
 * @stability unstable
 * @category decoding
 * @since 4.1.0
 */
export declare const decodeString: {
    /**
     * Creates a channel that parses text chunks into CSV records.
     *
     * **Details**
     *
     * Records are arrays of cells, or objects keyed by the first record when
     * `headers` is `true`. Parser state is created per channel run.
     *
     * **Example** (Parsing header records)
     *
     * ```ts import.meta.vitest
     * import { Effect, Stream } from "effect"
     * import { Csv } from "effect/encoding"
     *
     * const records = Stream.make("name,city\nAlice,\"San\nFrancisco\"\n").pipe(
     *   Stream.pipeThroughChannel(Csv.decodeString({ headers: true })),
     *   Stream.runCollect
     * )
     * await Effect.runPromise(records) // => [{ name: "Alice", city: "San\nFrancisco" }]
     * ```
     *
     * @see decode
     * @see decodeSchemaString
     * @stability unstable
     * @category decoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options: DecodeOptions & {
        readonly headers: true;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<Record.ReadonlyRecord<string, string>>, IE | CsvError, Done, Arr.NonEmptyReadonlyArray<string>, IE, Done>;
    /**
     * Creates a channel that parses text chunks into CSV records.
     *
     * **Details**
     *
     * Records are arrays of cells, or objects keyed by the first record when
     * `headers` is `true`. Parser state is created per channel run.
     *
     * **Example** (Parsing header records)
     *
     * ```ts import.meta.vitest
     * import { Effect, Stream } from "effect"
     * import { Csv } from "effect/encoding"
     *
     * const records = Stream.make("name,city\nAlice,\"San\nFrancisco\"\n").pipe(
     *   Stream.pipeThroughChannel(Csv.decodeString({ headers: true })),
     *   Stream.runCollect
     * )
     * await Effect.runPromise(records) // => [{ name: "Alice", city: "San\nFrancisco" }]
     * ```
     *
     * @see decode
     * @see decodeSchemaString
     * @stability unstable
     * @category decoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: DecodeOptions & {
        readonly headers?: false | undefined;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<Row>, IE | CsvError, Done, Arr.NonEmptyReadonlyArray<string>, IE, Done>;
    /**
     * Creates a channel that parses text chunks into CSV records.
     *
     * **Details**
     *
     * Records are arrays of cells, or objects keyed by the first record when
     * `headers` is `true`. Parser state is created per channel run.
     *
     * **Example** (Parsing header records)
     *
     * ```ts import.meta.vitest
     * import { Effect, Stream } from "effect"
     * import { Csv } from "effect/encoding"
     *
     * const records = Stream.make("name,city\nAlice,\"San\nFrancisco\"\n").pipe(
     *   Stream.pipeThroughChannel(Csv.decodeString({ headers: true })),
     *   Stream.runCollect
     * )
     * await Effect.runPromise(records) // => [{ name: "Alice", city: "San\nFrancisco" }]
     * ```
     *
     * @see decode
     * @see decodeSchemaString
     * @stability unstable
     * @category decoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: DecodeOptions): Channel.Channel<Arr.NonEmptyReadonlyArray<Row | Record.ReadonlyRecord<string, string>>, IE | CsvError, Done, Arr.NonEmptyReadonlyArray<string>, IE, Done>;
};
/**
 * Creates a channel that parses UTF-8 byte chunks into CSV records.
 *
 * @see decodeString
 * @stability unstable
 * @category decoding
 * @since 4.1.0
 */
export declare const decode: {
    /**
     * Creates a channel that parses UTF-8 byte chunks into CSV records.
     *
     * @see decodeString
     * @stability unstable
     * @category decoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options: DecodeOptions & {
        readonly headers: true;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<Record.ReadonlyRecord<string, string>>, IE | CsvError, Done, Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done>;
    /**
     * Creates a channel that parses UTF-8 byte chunks into CSV records.
     *
     * @see decodeString
     * @stability unstable
     * @category decoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: DecodeOptions & {
        readonly headers?: false | undefined;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<Row>, IE | CsvError, Done, Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done>;
    /**
     * Creates a channel that parses UTF-8 byte chunks into CSV records.
     *
     * @see decodeString
     * @stability unstable
     * @category decoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: DecodeOptions): Channel.Channel<Arr.NonEmptyReadonlyArray<Row | Record.ReadonlyRecord<string, string>>, IE | CsvError, Done, Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done>;
};
/**
 * Creates a channel that parses CSV text and decodes each record with a schema.
 *
 * **Details**
 *
 * With `headers: true` the schema receives objects keyed by header name;
 * otherwise it receives arrays of cells. Columns that do not encode to a string,
 * such as arrays or nested structs, throw when the channel is created. Each
 * chunk is decoded in one pass. Schema failures are `SchemaError`s whose issue
 * paths start with the record number.
 *
 * **Example** (Decoding renamed headers)
 *
 * ```ts import.meta.vitest
 * import { Effect, Schema, Stream } from "effect"
 * import { Csv } from "effect/encoding"
 *
 * const Person = Schema.Struct({
 *   name: Schema.String,
 *   age: Schema.Number
 * }).pipe(Schema.encodeKeys({ name: "Full Name", age: "Age (years)" }))
 *
 * const people = Stream.make("Full Name,Age (years)\nAlice,32").pipe(
 *   Stream.pipeThroughChannel(Csv.decodeSchemaString(Person)({ headers: true })),
 *   Stream.runCollect
 * )
 * await Effect.runPromise(people) // => [{ name: "Alice", age: 32 }]
 * ```
 *
 * **Example** (Transforming positional records into objects)
 *
 * ```ts import.meta.vitest
 * import { Effect, Schema, SchemaGetter, Stream } from "effect"
 * import { Csv } from "effect/encoding"
 *
 * const Person = Schema.Tuple([Schema.String, Schema.Number]).pipe(
 *   Schema.decodeTo(Schema.Struct({ name: Schema.String, age: Schema.Number }), {
 *     decode: SchemaGetter.transform(([name, age]) => ({ name, age })),
 *     encode: SchemaGetter.transform(({ name, age }) => [name, age] as const)
 *   })
 * )
 *
 * const people = Stream.make("Alice,32").pipe(
 *   Stream.pipeThroughChannel(Csv.decodeSchemaString(Person)()),
 *   Stream.runCollect
 * )
 * await Effect.runPromise(people) // => [{ name: "Alice", age: 32 }]
 * ```
 *
 * @see decodeSchema
 * @see encodeSchemaString
 * @stability unstable
 * @category decoding
 * @since 4.1.0
 */
export declare const decodeSchemaString: <S extends Schema.Constraint>(schema: S) => <IE = never, Done = unknown>(options?: DecodeSchemaOptions, parseOptions?: SchemaAST.ParseOptions) => Channel.Channel<Arr.NonEmptyReadonlyArray<S["Type"]>, IE | CsvError | Schema.SchemaError, Done, Arr.NonEmptyReadonlyArray<string>, IE, Done, S["DecodingServices"]>;
/**
 * Creates a channel that parses UTF-8 CSV and decodes each record with a schema.
 *
 * @see decodeSchemaString
 * @stability unstable
 * @category decoding
 * @since 4.1.0
 */
export declare const decodeSchema: <S extends Schema.Constraint>(schema: S) => <IE = never, Done = unknown>(options?: DecodeSchemaOptions, parseOptions?: SchemaAST.ParseOptions) => Channel.Channel<Arr.NonEmptyReadonlyArray<S["Type"]>, IE | CsvError | Schema.SchemaError, Done, Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done, S["DecodingServices"]>;
/**
 * Creates a channel that writes CSV records as text, one string per record.
 *
 * **Details**
 *
 * Without `columns`, records are arrays of cells. With `columns`, records are
 * objects encoded by those keys. A non-string cell is a defect.
 *
 * **Example** (Writing header records)
 *
 * ```ts import.meta.vitest
 * import { Effect, Stream } from "effect"
 * import { Csv } from "effect/encoding"
 *
 * const csv = Stream.make({ name: "Smith, Charlie", city: "London" }).pipe(
 *   Stream.pipeThroughChannel(Csv.encodeString({ columns: ["name", "city"], headers: true })),
 *   Stream.runCollect,
 *   Effect.map((parts) => parts.join(""))
 * )
 * await Effect.runPromise(csv) // => 'name,city\r\n"Smith, Charlie",London\r\n'
 * ```
 *
 * @see encode
 * @see encodeSchemaString
 * @stability unstable
 * @category encoding
 * @since 4.1.0
 */
export declare const encodeString: {
    /**
     * Creates a channel that writes CSV records as text, one string per record.
     *
     * **Details**
     *
     * Without `columns`, records are arrays of cells. With `columns`, records are
     * objects encoded by those keys. A non-string cell is a defect.
     *
     * **Example** (Writing header records)
     *
     * ```ts import.meta.vitest
     * import { Effect, Stream } from "effect"
     * import { Csv } from "effect/encoding"
     *
     * const csv = Stream.make({ name: "Smith, Charlie", city: "London" }).pipe(
     *   Stream.pipeThroughChannel(Csv.encodeString({ columns: ["name", "city"], headers: true })),
     *   Stream.runCollect,
     *   Effect.map((parts) => parts.join(""))
     * )
     * await Effect.runPromise(csv) // => 'name,city\r\n"Smith, Charlie",London\r\n'
     * ```
     *
     * @see encode
     * @see encodeSchemaString
     * @stability unstable
     * @category encoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options: EncodeOptions & {
        readonly columns: ReadonlyArray<string>;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<string>, IE, Done, Arr.NonEmptyReadonlyArray<Record.ReadonlyRecord<string, string | undefined>>, IE, Done>;
    /**
     * Creates a channel that writes CSV records as text, one string per record.
     *
     * **Details**
     *
     * Without `columns`, records are arrays of cells. With `columns`, records are
     * objects encoded by those keys. A non-string cell is a defect.
     *
     * **Example** (Writing header records)
     *
     * ```ts import.meta.vitest
     * import { Effect, Stream } from "effect"
     * import { Csv } from "effect/encoding"
     *
     * const csv = Stream.make({ name: "Smith, Charlie", city: "London" }).pipe(
     *   Stream.pipeThroughChannel(Csv.encodeString({ columns: ["name", "city"], headers: true })),
     *   Stream.runCollect,
     *   Effect.map((parts) => parts.join(""))
     * )
     * await Effect.runPromise(csv) // => 'name,city\r\n"Smith, Charlie",London\r\n'
     * ```
     *
     * @see encode
     * @see encodeSchemaString
     * @stability unstable
     * @category encoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: EncodeOptions & {
        readonly columns?: undefined;
        readonly headers?: false | undefined;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<string>, IE, Done, Arr.NonEmptyReadonlyArray<Row>, IE, Done>;
    /**
     * Creates a channel that writes CSV records as text, one string per record.
     *
     * **Details**
     *
     * Without `columns`, records are arrays of cells. With `columns`, records are
     * objects encoded by those keys. A non-string cell is a defect.
     *
     * **Example** (Writing header records)
     *
     * ```ts import.meta.vitest
     * import { Effect, Stream } from "effect"
     * import { Csv } from "effect/encoding"
     *
     * const csv = Stream.make({ name: "Smith, Charlie", city: "London" }).pipe(
     *   Stream.pipeThroughChannel(Csv.encodeString({ columns: ["name", "city"], headers: true })),
     *   Stream.runCollect,
     *   Effect.map((parts) => parts.join(""))
     * )
     * await Effect.runPromise(csv) // => 'name,city\r\n"Smith, Charlie",London\r\n'
     * ```
     *
     * @see encode
     * @see encodeSchemaString
     * @stability unstable
     * @category encoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: EncodeOptions): Channel.Channel<Arr.NonEmptyReadonlyArray<string>, IE, Done, Arr.NonEmptyReadonlyArray<Row | Record.ReadonlyRecord<string, string | undefined>>, IE, Done>;
};
/**
 * Creates a channel that writes CSV records as UTF-8 bytes, one byte array per
 * input chunk.
 *
 * @see encodeString
 * @stability unstable
 * @category encoding
 * @since 4.1.0
 */
export declare const encode: {
    /**
     * Creates a channel that writes CSV records as UTF-8 bytes, one byte array per
     * input chunk.
     *
     * @see encodeString
     * @stability unstable
     * @category encoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options: EncodeOptions & {
        readonly columns: ReadonlyArray<string>;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done, Arr.NonEmptyReadonlyArray<Record.ReadonlyRecord<string, string | undefined>>, IE, Done>;
    /**
     * Creates a channel that writes CSV records as UTF-8 bytes, one byte array per
     * input chunk.
     *
     * @see encodeString
     * @stability unstable
     * @category encoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: EncodeOptions & {
        readonly columns?: undefined;
        readonly headers?: false | undefined;
    }): Channel.Channel<Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done, Arr.NonEmptyReadonlyArray<Row>, IE, Done>;
    /**
     * Creates a channel that writes CSV records as UTF-8 bytes, one byte array per
     * input chunk.
     *
     * @see encodeString
     * @stability unstable
     * @category encoding
     * @since 4.1.0
     */
    <IE = never, Done = unknown>(options?: EncodeOptions): Channel.Channel<Arr.NonEmptyReadonlyArray<Uint8Array>, IE, Done, Arr.NonEmptyReadonlyArray<Row | Record.ReadonlyRecord<string, string | undefined>>, IE, Done>;
};
/**
 * Creates a channel that encodes values with a schema and writes them as CSV
 * text, one string per record.
 *
 * **Details**
 *
 * Columns default to the encoded keys of a struct schema, including keys
 * renamed with `Schema.encodeKeys`. Missing or `undefined` values become empty
 * cells. Schema failures are `SchemaError`s whose issue paths start with the
 * output record number, counting the header row.
 *
 * **Example** (Encoding renamed headers)
 *
 * ```ts import.meta.vitest
 * import { Effect, Schema, Stream } from "effect"
 * import { Csv } from "effect/encoding"
 *
 * const Person = Schema.Struct({
 *   name: Schema.String,
 *   age: Schema.Number,
 *   joined: Schema.Date
 * }).pipe(Schema.encodeKeys({ name: "Full Name", age: "Age (years)" }))
 *
 * const csv = await Effect.runPromise(
 *   Stream.make({ name: "Alice", age: 32, joined: new Date(0) }).pipe(
 *     Stream.pipeThroughChannel(Csv.encodeSchemaString(Person)({ headers: true, newline: "\n" })),
 *     Stream.runCollect,
 *     Effect.map((parts) => parts.join(""))
 *   )
 * )
 * csv.split("\n") // => ["Full Name,Age (years),joined", "Alice,32,1970-01-01T00:00:00.000Z", ""]
 * ```
 *
 * @see decodeSchemaString
 * @see encodeSchema
 * @stability unstable
 * @category encoding
 * @since 4.1.0
 */
export declare const encodeSchemaString: <S extends Schema.Constraint>(schema: S) => <IE = never, Done = unknown>(options?: EncodeSchemaOptions<S>) => Channel.Channel<Arr.NonEmptyReadonlyArray<string>, IE | Schema.SchemaError, Done, Arr.NonEmptyReadonlyArray<S["Type"]>, IE, Done, S["EncodingServices"]>;
/**
 * Creates a channel that encodes values with a schema and writes them as UTF-8
 * CSV bytes.
 *
 * **Gotchas**
 *
 * When the bytes stream an HTTP response body, for example through
 * `HttpApiSchema.StreamUint8Array`, an encoding failure ends the body; it
 * cannot change a status that was already sent.
 *
 * @see encodeSchemaString
 * @stability unstable
 * @category encoding
 * @since 4.1.0
 */
export declare const encodeSchema: <S extends Schema.Constraint>(schema: S) => <IE = never, Done = unknown>(options?: EncodeSchemaOptions<S>) => Channel.Channel<Arr.NonEmptyReadonlyArray<Uint8Array>, IE | Schema.SchemaError, Done, Arr.NonEmptyReadonlyArray<S["Type"]>, IE, Done, S["EncodingServices"]>;
export {};
//# sourceMappingURL=Csv.d.ts.map