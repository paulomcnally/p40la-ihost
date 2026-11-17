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
import * as Arr from "../Array.js";
import * as Cause from "../Cause.js";
import * as Channel from "../Channel.js";
import * as Data from "../Data.js";
import * as Effect from "../Effect.js";
import * as Predicate from "../Predicate.js";
import * as Pull from "../Pull.js";
import * as Schema from "../Schema.js";
import * as SchemaAST from "../SchemaAST.js";
import * as SchemaIssue from "../SchemaIssue.js";
const CsvErrorTypeId = "~effect/encoding/Csv/CsvError";
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
export class CsvError extends /*#__PURE__*/Data.TaggedError("CsvError") {
  /**
   * Identifies CSV errors across module instances.
   *
   * @stability unstable
   * @since 4.1.0
   */
  [CsvErrorTypeId] = CsvErrorTypeId;
}
/**
 * Checks whether a value is a `CsvError`.
 *
 * @stability unstable
 * @category guards
 * @since 4.1.0
 */
export const isCsvError = u => Predicate.hasProperty(u, CsvErrorTypeId);
const defaultMaxRecordSize = 1024 * 1024;
const checkDelimiter = delimiter => {
  if (delimiter.length !== 1 || delimiter === "\"" || delimiter === "\r" || delimiter === "\n") {
    throw new Error(`Csv: delimiter must be one character other than a quote or line break, got ${delimiter}`);
  }
  return delimiter;
};
const QUOTE = 34;
const CR = 13;
const LF = 10;
// Parser states. QUOTE_IN_QUOTED follows a quote inside a quoted field, which
// either closes the field or starts an escaped quote.
const FIELD_START = 0;
const UNQUOTED = 1;
const QUOTED = 2;
const QUOTE_IN_QUOTED = 3;
class Parser {
  delimiter;
  maxRecordSize;
  ignoreEmptyLines;
  state = FIELD_START;
  // The partial field, carried across chunks.
  field = "";
  cells = [];
  // The size of the current record before the current chunk.
  size = 0;
  record = 1;
  line = 1;
  recordLine = 1;
  started = false;
  // A CR ended the previous record, so a leading LF completes a CRLF.
  skipLF = false;
  // Quoted content ended with a CR, so a leading LF does not start a new line.
  quotedCR = false;
  constructor(delimiter, maxRecordSize, ignoreEmptyLines) {
    this.delimiter = delimiter.charCodeAt(0);
    this.maxRecordSize = maxRecordSize;
    this.ignoreEmptyLines = ignoreEmptyLines;
  }
  fail(kind, message) {
    return new CsvError({
      kind,
      message,
      record: this.record,
      line: this.recordLine,
      column: kind === "Syntax" ? this.cells.length + 1 : undefined
    });
  }
  checkSize(size) {
    if (size > this.maxRecordSize) {
      throw this.fail("RecordSize", `CSV record exceeds maxRecordSize of ${this.maxRecordSize}`);
    }
  }
  // A record that is already too large reports its size, not the syntax error.
  failSyntax(size, message) {
    this.checkSize(size);
    return this.fail("Syntax", message);
  }
  endRecord(size, out) {
    this.checkSize(size);
    if (size > 0 || !this.ignoreEmptyLines) {
      out.push({
        cells: this.cells,
        record: this.record,
        line: this.recordLine
      });
    }
    this.cells = [];
    this.size = 0;
    this.record++;
  }
  countLines(text, start, end) {
    for (let i = start; i < end; i++) {
      const c = text.charCodeAt(i);
      if (c === CR) this.line++;else if (c === LF && !(i === start ? this.quotedCR : text.charCodeAt(i - 1) === CR)) this.line++;
    }
    if (end > start) this.quotedCR = text.charCodeAt(end - 1) === CR;
  }
  feed(text, out) {
    const length = text.length;
    const delimiter = this.delimiter;
    let i = 0;
    if (length > 0 && !this.started) {
      this.started = true;
      if (text.charCodeAt(0) === 0xFEFF) i = 1;
    }
    // The current record starts at `start` in this chunk, so its size at `i`
    // is `this.size + i - start`.
    let start = i;
    let state = this.state;
    let field = this.field;
    while (i < length) {
      if (state === FIELD_START) {
        if (this.skipLF) {
          this.skipLF = false;
          if (text.charCodeAt(i) === LF) {
            start = ++i;
            continue;
          }
        }
        if (text.charCodeAt(i) === QUOTE) {
          state = QUOTED;
          this.quotedCR = false;
          i++;
          continue;
        }
        state = UNQUOTED;
      }
      if (state === UNQUOTED) {
        let j = i;
        let c = 0;
        while (j < length) {
          c = text.charCodeAt(j);
          if (c === delimiter || c === CR || c === LF || c === QUOTE) break;
          j++;
        }
        if (j > i) field = field === "" ? text.slice(i, j) : field + text.slice(i, j);
        if (j === length) {
          i = j;
          break;
        }
        if (c === QUOTE) {
          throw this.failSyntax(this.size + j - start, "Unexpected quote in an unquoted field");
        }
        this.cells.push(field);
        field = "";
        state = FIELD_START;
        i = j + 1;
        if (c !== delimiter) {
          this.endRecord(this.size + j - start, out);
          this.line++;
          this.recordLine = this.line;
          this.skipLF = c === CR;
          start = i;
        }
        continue;
      }
      if (state === QUOTED) {
        const end = text.indexOf("\"", i);
        const j = end === -1 ? length : end;
        if (j > i) {
          field += text.slice(i, j);
          this.countLines(text, i, j);
        }
        if (end === -1) {
          i = j;
          break;
        }
        state = QUOTE_IN_QUOTED;
        i = j + 1;
        continue;
      }
      // QUOTE_IN_QUOTED
      const c = text.charCodeAt(i);
      if (c === QUOTE) {
        field += "\"";
        state = QUOTED;
        this.quotedCR = false;
        i++;
      } else if (c === delimiter || c === CR || c === LF) {
        this.cells.push(field);
        field = "";
        state = FIELD_START;
        i++;
        if (c !== delimiter) {
          this.endRecord(this.size + i - 1 - start, out);
          this.line++;
          this.recordLine = this.line;
          this.skipLF = c === CR;
          start = i;
        }
      } else {
        throw this.failSyntax(this.size + i - start, "Expected a delimiter or line break after a closing quote");
      }
    }
    this.state = state;
    this.field = field;
    this.size += length - start;
    this.checkSize(this.size);
  }
  end(out) {
    if (this.state === QUOTED) throw this.fail("Syntax", "Unterminated quoted field");
    if (this.size > 0 || this.cells.length > 0) {
      this.cells.push(this.field);
      this.field = "";
      this.state = FIELD_START;
      this.endRecord(this.size, out);
    }
  }
}
const decodeRecords = options => {
  const delimiter = checkDelimiter(options?.delimiter ?? ",");
  const maxRecordSize = options?.maxRecordSize ?? defaultMaxRecordSize;
  if (!(maxRecordSize > 0)) {
    throw new Error(`Csv: maxRecordSize must be positive, got ${maxRecordSize}`);
  }
  const ignoreEmptyLines = options?.ignoreEmptyLines === true;
  const useHeaders = options?.headers === true;
  return Channel.fromTransform((upstream, _scope) => Effect.sync(() => {
    const parser = new Parser(delimiter, maxRecordSize, ignoreEmptyLines);
    let headers;
    // Plain assignment is faster than `Object.fromEntries`, but a `__proto__`
    // header would set the prototype instead of a property.
    let assignable = false;
    let done;
    const toRecord = row => {
      if (!useHeaders) return {
        value: row.cells,
        record: row.record
      };
      if (row.cells.length !== headers.length) {
        throw new CsvError({
          kind: "Header",
          message: `Expected ${headers.length} cells, got ${row.cells.length}`,
          record: row.record,
          line: row.line,
          column: undefined
        });
      }
      if (!assignable) {
        return {
          value: Object.fromEntries(headers.map((name, i) => [name, row.cells[i]])),
          record: row.record
        };
      }
      const value = {};
      for (let i = 0; i < headers.length; i++) value[headers[i]] = row.cells[i];
      return {
        value,
        record: row.record
      };
    };
    const emit = run => {
      const rows = [];
      const out = [];
      try {
        run(rows);
        for (const row of rows) {
          if (useHeaders && headers === undefined) {
            if (new Set(row.cells).size !== row.cells.length) {
              throw new CsvError({
                kind: "Header",
                message: "Duplicate header name",
                record: row.record,
                line: row.line,
                column: undefined
              });
            }
            headers = row.cells;
            assignable = !headers.includes("__proto__");
          } else {
            out.push(toRecord(row));
          }
        }
      } catch (error) {
        if (isCsvError(error)) return Effect.fail(error);
        throw error;
      }
      return Arr.isArrayNonEmpty(out) ? Effect.succeed(out) : pull;
    };
    const pull = Effect.suspend(() => {
      if (done !== undefined) return done;
      return Pull.matchEffect(upstream, {
        onSuccess: texts => emit(out => {
          for (const text of texts) parser.feed(text, out);
        }),
        onFailure: Effect.failCause,
        onDone: value => {
          done = Cause.done(value);
          return emit(out => parser.end(out));
        }
      });
    });
    return pull;
  }));
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
export const decodeString = options => Channel.map(decodeRecords(options), Arr.map(record => record.value));
/**
 * Creates a channel that parses UTF-8 byte chunks into CSV records.
 *
 * @see decodeString
 * @stability unstable
 * @category decoding
 * @since 4.1.0
 */
export const decode = options => Channel.pipeTo(Channel.decodeText("utf-8", {
  ignoreBOM: true
}), decodeString(options));
// Rewrites the chunk index at the head of each top-level issue path to the
// record number, keeping the rest of the issue unchanged.
const rebaseIssue = (error, record) => {
  const issue = error.issue;
  if (issue._tag !== "Composite") return error;
  return new Schema.SchemaError(new SchemaIssue.Composite(issue.ast, Arr.map(issue.issues, issue => issue._tag === "Pointer" && typeof issue.path[0] === "number" ? new SchemaIssue.Pointer([record(issue.path[0]), ...issue.path.slice(1)], issue.issue) : issue), issue.input, {
    reportInput: "input" in issue
  }));
};
// Derived codecs are new ASTs, and schema parsers are cached per AST, so they
// are memoized to avoid recompiling parsers for every channel.
const stringTreeCodecs = /*#__PURE__*/new WeakMap();
const arrayCodecs = /*#__PURE__*/new WeakMap();
// The schema used for CSV cells. `toCodecStringTree` keeps the decoded type and
// services and only changes the encoded side, which is treated as unknown here.
const toCodec = (schema, disableCodecs) => {
  if (disableCodecs === true) return schema;
  let codec = stringTreeCodecs.get(schema);
  if (codec === undefined) {
    codec = Schema.toCodecStringTree(schema);
    stringTreeCodecs.set(schema, codec);
  }
  return codec;
};
const toArrayCodec = codec => {
  let array = arrayCodecs.get(codec);
  if (array === undefined) {
    array = Schema.NonEmptyArray(codec);
    arrayCodecs.set(codec, array);
  }
  return array;
};
const isStringLeaf = ast => {
  switch (ast._tag) {
    case "String":
    case "TemplateLiteral":
      return true;
    case "Literal":
      return typeof ast.literal === "string";
    case "Enum":
      return ast.enums.every(([, value]) => typeof value === "string");
    case "Union":
      return ast.types.every(type => type._tag === "Undefined" || isStringLeaf(type));
    default:
      return false;
  }
};
const resolveSuspend = ast => SchemaAST.isSuspend(ast) ? resolveSuspend(ast.thunk()) : ast;
// Throws unless every selected cell of an encoded struct or tuple is a string.
const checkCells = (ast, columns) => {
  ast = resolveSuspend(ast);
  const check = (name, type) => {
    if (!isStringLeaf(type)) {
      throw new Error(`Csv: ${name} encodes to ${type._tag}, not a string. ` + "Encode it as a single cell with a transformation, such as SchemaGetter.split for arrays");
    }
  };
  if (SchemaAST.isObjects(ast)) {
    for (const ps of ast.propertySignatures) {
      if (typeof ps.name === "string" && (columns === undefined || columns.includes(ps.name))) {
        check(`column ${JSON.stringify(ps.name)}`, ps.type);
      }
    }
    for (const is of ast.indexSignatures) check("index signature", is.type);
  } else if (SchemaAST.isArrays(ast)) {
    ast.elements.forEach((type, i) => check(`column ${i + 1}`, type));
    for (const type of ast.rest) check("rest element", type);
  } else if (SchemaAST.isUnion(ast)) {
    for (const type of ast.types) checkCells(type, columns);
  } else {
    throw new Error(`Csv: cannot verify string cells for ${ast._tag}. ` + "Encode it as a flat struct or tuple with a transformation");
  }
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
export const decodeSchemaString = schema => (options, parseOptions) => {
  const codec = toCodec(schema, options?.disableCodecs);
  const ast = resolveSuspend(SchemaAST.toEncoded(codec.ast));
  checkCells(ast, undefined);
  // CSV cannot tell an empty cell from a missing one, so empty cells of
  // optional properties are decoded as missing keys.
  const optional = options?.disableCodecs !== true && options?.headers === true && SchemaAST.isObjects(ast) ? ast.propertySignatures.flatMap(ps => typeof ps.name === "string" && SchemaAST.isOptional(ps.type) ? [ps.name] : []) : [];
  const input = record => {
    if (optional.length === 0) return record.value;
    // Header records are created per row, so they can be updated in place.
    const value = record.value;
    for (const key of optional) {
      if (value[key] === "") delete value[key];
    }
    return value;
  };
  const decode = Schema.decodeUnknownEffect(toArrayCodec(codec), parseOptions);
  return Channel.mapEffect(decodeRecords(options), records => Effect.mapError(decode(Arr.map(records, input)), error => rebaseIssue(error, index => records[index].record)));
};
/**
 * Creates a channel that parses UTF-8 CSV and decodes each record with a schema.
 *
 * @see decodeSchemaString
 * @stability unstable
 * @category decoding
 * @since 4.1.0
 */
export const decodeSchema = schema => (options, parseOptions) => Channel.pipeTo(Channel.decodeText("utf-8", {
  ignoreBOM: true
}), decodeSchemaString(schema)(options, parseOptions));
const textEncoder = /*#__PURE__*/new TextEncoder();
const makeWriter = options => {
  const delimiter = checkDelimiter(options?.delimiter ?? ",");
  const newline = options?.newline ?? "\r\n";
  if (newline !== "\n" && newline !== "\r\n") {
    throw new Error(`Csv: newline must be "\\n" or "\\r\\n", got ${JSON.stringify(newline)}`);
  }
  const columns = options?.columns;
  if (options?.headers === true && columns === undefined) {
    throw new Error("Csv: headers requires columns");
  }
  // Characters that require a cell to be quoted.
  const special = new RegExp(`[${delimiter.replace(/[\\\]^-]/g, "\\$&")}"\\r\\n]`);
  const cell = value => {
    if (value === undefined) return "";
    if (typeof value !== "string") throw new TypeError(`Csv: expected a string cell, got ${typeof value}`);
    // A leading BOM is quoted so that decoders do not strip it.
    return special.test(value) || value.charCodeAt(0) === 0xFEFF ? "\"" + value.replaceAll("\"", "\"\"") + "\"" : value;
  };
  const write = cells => {
    // A single empty cell is quoted so that it is not read back as an empty line.
    if (cells.length === 1 && (cells[0] === "" || cells[0] === undefined)) return "\"\"" + newline;
    let line = cell(cells[0]);
    for (let i = 1; i < cells.length; i++) line += delimiter + cell(cells[i]);
    return line + newline;
  };
  const header = options?.headers === true ? write(columns) : undefined;
  const writeRecord = value => {
    if (Array.isArray(value)) return write(value);
    if (columns === undefined) throw new TypeError("Csv: encoding object records requires columns");
    const record = value;
    return write(columns.map(key => Object.hasOwn(record, key) ? record[key] : undefined));
  };
  return {
    header,
    writeRecord
  };
};
const encodeRecords = options => {
  const {
    header,
    writeRecord
  } = makeWriter(options);
  return Channel.fromTransform((upstream, _scope) => Effect.sync(() => {
    let pending = header;
    const pull = Effect.suspend(() => {
      if (pending !== undefined) {
        const text = pending;
        pending = undefined;
        return Effect.succeed(Arr.of(text));
      }
      return Effect.flatMap(upstream, records => Effect.sync(() => Arr.map(records, writeRecord)));
    });
    return pull;
  }));
};
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
export const encodeString = options => encodeRecords(options);
/**
 * Creates a channel that writes CSV records as UTF-8 bytes, one byte array per
 * input chunk.
 *
 * @see encodeString
 * @stability unstable
 * @category encoding
 * @since 4.1.0
 */
export const encode = options => Channel.map(encodeRecords(options), texts => Arr.of(textEncoder.encode(texts.join(""))));
// Struct schemas encode by their encoded keys in declaration order. Tuple and
// array schemas encode by position. Other schemas, including records with
// index signatures, need explicit columns.
const schemaColumns = (ast, options) => {
  if (options?.columns !== undefined) {
    checkCells(ast, options.columns);
    return options.columns;
  }
  if (SchemaAST.isObjects(ast) && ast.indexSignatures.length === 0) {
    const columns = ast.propertySignatures.flatMap(ps => typeof ps.name === "string" ? [ps.name] : []);
    checkCells(ast, columns);
    return columns;
  }
  if (SchemaAST.isArrays(ast)) {
    if (options?.headers === true) throw new Error("Csv: headers for tuple or array schemas require columns");
    checkCells(ast, undefined);
    return undefined;
  }
  throw new Error(`Csv: columns are required to encode ${ast._tag} schemas`);
};
const encodeSchemaRecords = (schema, options) => {
  const codec = toCodec(schema, options?.disableCodecs);
  const columns = schemaColumns(SchemaAST.toEncoded(codec.ast), options);
  const writer = encodeRecords({
    ...options,
    columns
  });
  const encode = Schema.encodeEffect(toArrayCodec(codec));
  const firstRecord = options?.headers === true ? 2 : 1;
  const values = Channel.fromTransform((upstream, _scope) => Effect.sync(() => {
    let record = firstRecord;
    return Effect.flatMap(upstream, chunk => {
      const first = record;
      record += chunk.length;
      return Effect.mapError(encode(chunk), error => rebaseIssue(error, index => first + index));
    });
  }));
  return Channel.pipeTo(values, writer);
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
export const encodeSchemaString = schema => options => encodeSchemaRecords(schema, options);
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
export const encodeSchema = schema => options => Channel.map(encodeSchemaRecords(schema, options), texts => Arr.of(textEncoder.encode(texts.join(""))));
//# sourceMappingURL=Csv.js.map