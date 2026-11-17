/**
 * Defines endpoint declarations used inside an HTTP API group.
 *
 * An endpoint records a stable identifier, HTTP method, router path, request schemas,
 * response schemas, declared errors, middleware, and annotations. Endpoint
 * values are declarations, not handlers: builders use them to decode requests,
 * type handler input, encode responses, generate OpenAPI metadata, and derive
 * generated-client call signatures. This module also includes HTTP method
 * constructors, payload and response schema helpers, and type utilities used by
 * builders and generated clients.
 *
 * **Request variants**
 *
 * A server needs a distinct method and path per endpoint.
 * `HttpApiBuilder.layer` rejects two endpoints that share them, even when their
 * query, headers or payload differ. Declare the variants of one operation as a
 * single endpoint instead:
 *
 * - discriminate with a `query` or `headers` field, such as `stream: Schema.Boolean`
 * - list several `payload` schemas, chosen by the request `Content-Type`
 * - list several `success` schemas with distinct content types or statuses. One
 *   of them can be a stream. The server encodes whichever value the handler
 *   returns, and the client decodes by the response `Content-Type`
 *
 * The handler branches on the decoded request, and OpenAPI documents one
 * operation with every alternative. The generated client returns the union of
 * the success types. Wrap the client method to narrow the result per variant
 * when callers need it.
 *
 * **Example** (Buffered and streaming variants of one operation)
 *
 * ```ts import.meta.vitest
 * import { Effect, Schema, Stream } from "effect"
 * import { HttpApi, HttpApiBuilder, HttpApiEndpoint, HttpApiGroup, HttpApiSchema } from "effect/http-api"
 *
 * const Stats = Schema.Struct({ cpu: Schema.Number })
 *
 * const Api = HttpApi.make("Docker").add(
 *   HttpApiGroup.make("containers").add(
 *     // GET /stats?stream=false returns JSON, ?stream=true returns NDJSON
 *     HttpApiEndpoint.get("stats", "/stats", {
 *       query: { stream: Schema.Boolean },
 *       success: [Stats, HttpApiSchema.StreamNdjson({ schema: Stats })]
 *     }),
 *     // POST /images accepts a JSON pull request or a tar archive
 *     HttpApiEndpoint.post("createImage", "/images", {
 *       payload: [
 *         Schema.Struct({ fromImage: Schema.String }),
 *         HttpApiSchema.StreamUint8Array({ contentType: "application/x-tar" })
 *       ],
 *       success: Schema.String
 *     })
 *   )
 * )
 *
 * const ContainersLive = HttpApiBuilder.group(Api, "containers", (handlers) =>
 *   handlers
 *     .handle("stats", ({ query }) =>
 *       Effect.succeed(query.stream ? Stream.make({ cpu: 1 }, { cpu: 2 }) : { cpu: 1 })
 *     )
 *     .handle("createImage", ({ payload }) =>
 *       Stream.isStream(payload)
 *         ? Effect.as(Stream.runDrain(payload), "imported")
 *         : Effect.succeed(`pulled ${payload.fromImage}`)
 *     ))
 * ```
 *
 * @stability unstable
 * @since 4.0.0
 */
import * as Arr from "../Array.js";
import * as Context from "../Context.js";
import { identity, memoize } from "../Function.js";
import { hasBody } from "../http/HttpMethod.js";
import * as HttpRouter from "../http/HttpRouter.js";
import { pipeArguments } from "../Pipeable.js";
import * as Predicate from "../Predicate.js";
import * as Schema from "../Schema.js";
import * as AST from "../SchemaAST.js";
import * as HttpApiSchema from "./HttpApiSchema.js";
import * as MediaType from "./internal/mediaType.js";
import { toCodecHttpProperties } from "./internal/toCodecHttpProperties.js";
const TypeId = "~effect/http-api/HttpApiEndpoint";
/**
 * Returns `true` when a value is an `HttpApiEndpoint`, narrowing the value to the
 * endpoint interface.
 *
 * @stability unstable
 * @category guards
 * @since 4.0.0
 */
export const isHttpApiEndpoint = u => Predicate.hasProperty(u, TypeId);
/**
 * @internal
 */
export function getPayloadSchemas(endpoint) {
  const result = [];
  for (const {
    schemas
  } of endpoint.payload.values()) {
    result.push(...schemas);
  }
  return result;
}
/**
 * @internal
 */
export function getSuccessSchemas(endpoint) {
  const schemas = Array.from(endpoint.success);
  return Arr.isArrayNonEmpty(schemas) ? schemas : [HttpApiSchema.NoContent];
}
/**
 * @internal
 */
export function getErrorSchemas(endpoint) {
  const schemas = new Set(endpoint.error);
  const transform = endpoint.disableCodecs ? identity : transformResponseSchema;
  for (const middleware of endpoint.middlewares) {
    const key = middleware;
    for (const schema of key.error) {
      schemas.add(transform(schema));
    }
  }
  return Array.from(schemas);
}
const Proto = {
  [TypeId]: TypeId,
  pipe() {
    return pipeArguments(this, arguments);
  },
  prefix(prefix) {
    return makeProto({
      ...optionsFromEndpoint(this),
      path: HttpRouter.prefixPath(this.path, prefix)
    });
  },
  middleware(middleware) {
    return makeProto({
      ...optionsFromEndpoint(this),
      middlewares: new Set([...this.middlewares, middleware])
    });
  },
  annotate(key, value) {
    return makeProto({
      ...optionsFromEndpoint(this),
      annotations: Context.add(this.annotations, key, value)
    });
  },
  annotateMerge(annotations) {
    return makeProto({
      ...optionsFromEndpoint(this),
      annotations: Context.merge(this.annotations, annotations)
    });
  }
};
const optionsFromEndpoint = endpoint => ({
  identifier: endpoint.identifier,
  path: endpoint.path,
  method: endpoint.method,
  params: endpoint.params,
  query: endpoint.query,
  headers: endpoint.headers,
  payload: endpoint.payload,
  success: endpoint.success,
  error: endpoint.error,
  annotations: endpoint.annotations,
  middlewares: endpoint.middlewares,
  disableCodecs: endpoint.disableCodecs
});
function makeProto(options) {
  function HttpApiEndpoint() {}
  Object.setPrototypeOf(HttpApiEndpoint, Proto);
  return Object.assign(HttpApiEndpoint, options);
}
/**
 * Creates endpoint constructors for a specific HTTP method. The resulting
 * constructor builds an `HttpApiEndpoint` from an identifier, path, and optional request
 * and response schemas, applying automatic JSON or string-tree codecs unless
 * `disableCodecs` is enabled.
 *
 * **Gotchas**
 *
 * URL parameters are flat on the server: each `query` field, each field of a
 * no-body `payload` and each field of a form-urlencoded `payload` should encode
 * to a string or an array of strings. A field that encodes to a nested object
 * is sent as bracket keys (`meta[k]=v`) that the server does not read back, so
 * it is dropped, or reported missing when required. Send structured values with
 * an explicit codec such as `Schema.fromJsonString(...)`, or declare bracket
 * keys literally, for example `"page[size]": Schema.String`.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const make = method => (identifier, path, options) => {
  const disableCodecs = options?.disableCodecs ?? false;
  const transformProperties = disableCodecs ? identity : toCodecHttpProperties;
  const payload = getPayload(options?.payload, method, disableCodecs);
  return makeProto({
    identifier,
    path,
    method,
    params: ensureStruct(options?.params, transformProperties),
    query: ensureStruct(options?.query, transformProperties),
    headers: ensureStruct(options?.headers, transformProperties),
    payload,
    success: getSuccessResponse(options?.success, method, payload, disableCodecs),
    error: getErrorResponse(options?.error, disableCodecs),
    annotations: Context.empty(),
    middlewares: new Set(),
    disableCodecs
  });
};
function ensureStruct(params, transform) {
  if (params === undefined) return undefined;
  if (Schema.isSchema(params)) return transform(params);
  return transform(Schema.Struct(params));
}
function getPayload(payload, method, disableCodecs) {
  const result = new Map();
  if (payload === undefined) return result;
  const schemas = Array.isArray(payload) ? payload : Schema.isSchema(payload) ? [payload] : [Schema.Struct(payload).pipe(HttpApiSchema.asFormUrlEncoded())];
  const transform = disableCodecs ? identity : transformPayload;
  let hasStream = false;
  for (const schema of schemas) {
    if (HttpApiSchema.isStreamSchema(schema)) {
      if (HttpApiSchema.isStreamSse(schema)) {
        throw new Error("StreamSse is response-only and cannot be used as a payload");
      }
      if (!hasBody(method)) {
        throw new Error(`${method} endpoints cannot declare a streaming payload`);
      }
      if (hasStream) {
        throw new Error("Multiple streaming payloads are not supported");
      }
      hasStream = true;
      for (const declared of schema.contentTypes) {
        const contentType = MediaType.normalize(declared);
        if (result.has(contentType)) {
          throw new Error(`Cannot combine buffered and streaming payloads for content-type: ${declared}`);
        }
        result.set(contentType, {
          encoding: {
            _tag: "Stream",
            contentType: declared
          },
          schemas: [schema]
        });
      }
      continue;
    }
    const encoding = HttpApiSchema.getPayloadEncoding(schema.ast, method);
    const contentType = MediaType.normalize(encoding.contentType);
    const existing = result.get(contentType);
    if (existing) {
      if (existing.encoding._tag === "Stream") {
        throw new Error(`Cannot combine buffered and streaming payloads for content-type: ${encoding.contentType}`);
      }
      if (existing.encoding._tag !== encoding._tag) {
        throw new Error(`Multiple payload encodings for content-type: ${encoding.contentType}`);
      }
      if (existing.encoding._tag === "Multipart") {
        throw new Error(`Multiple multipart payloads for content-type: ${encoding.contentType}`);
      }
      existing.schemas.push(transform(schema, method));
    } else {
      result.set(contentType, {
        encoding,
        schemas: [transform(schema, method)]
      });
    }
  }
  return result;
}
const reservedStreamFailureEvent = "effect/http-api/stream/failure";
function getSuccessResponse(success, method, payload, disableCodecs) {
  if (success === undefined) return new Set();
  const schemas = Arr.ensure(success);
  validateSuccessResponse(schemas, method, payload);
  return new Set(disableCodecs ? schemas : schemas.map(transformResponseSchema));
}
const transformResponseSchema = /*#__PURE__*/memoize(schema => {
  if (HttpApiSchema.isStreamSchema(schema)) return schema;
  if (HttpApiSchema.isUpgrade(schema)) {
    return schema.headers === undefined ? schema : HttpApiSchema.rebuildUpgrade(schema.protocol, toCodecHttpProperties(schema.headers));
  }
  if (HttpApiSchema.isWithHeaders(schema)) {
    const inner = HttpApiSchema.isStreamSchema(schema.schema) ? schema.schema : applyResponseEncoding(schema.schema, HttpApiSchema.getResponseEncodingSchema(schema));
    return HttpApiSchema.rebuildWithHeaders(schema, inner, toCodecHttpProperties(schema.headers));
  }
  return transformResponse(schema);
});
function getErrorResponse(error, disableCodecs) {
  if (error === undefined) return new Set();
  const schemas = Arr.ensure(error);
  for (const schema of schemas) {
    const body = HttpApiSchema.isWithHeaders(schema) ? schema.schema : schema;
    if (HttpApiSchema.isStreamSchema(body)) {
      throw new Error("Streaming schemas are not supported in error responses");
    }
    if (HttpApiSchema.isUpgrade(body)) {
      throw new Error("Upgrade schemas are not supported in error responses");
    }
  }
  validateResponseExclusivity(schemas, HttpApiSchema.getStatusErrorSchema);
  return new Set(disableCodecs ? schemas : schemas.map(transformResponseSchema));
}
function validateSuccessResponse(schemas, method, payload) {
  const upgrades = schemas.filter(HttpApiSchema.isUpgrade);
  if (upgrades.length > 1) {
    throw new Error("Cannot declare more than one upgrade");
  }
  const hasUpgrade = upgrades.length === 1;
  if (hasUpgrade) {
    if (method === "HEAD") {
      throw new Error("HEAD endpoints cannot declare an upgrade");
    }
    // The 101 can only follow the end of the request, so the body is read before the upgrade.
    for (const {
      encoding
    } of payload.values()) {
      if (encoding._tag === "Stream" || encoding._tag === "Multipart" && encoding.mode === "stream") {
        throw new Error("Upgrade endpoints cannot declare a streaming payload");
      }
    }
  }
  let hasStream = false;
  const statuses = new Map();
  for (const schema of schemas) {
    // The server may decline the upgrade (RFC 9110 §7.8) and answer with one of the other successes.
    if (HttpApiSchema.isUpgrade(schema)) continue;
    const inner = HttpApiSchema.isWithHeaders(schema) ? schema.schema : schema;
    const status = HttpApiSchema.getStatusSuccessSchema(schema);
    if (hasUpgrade && status === 101) {
      throw new Error("Cannot combine an upgrade with another success response for status: 101");
    }
    if (HttpApiSchema.isStreamSchema(inner)) {
      if (hasUpgrade) {
        throw new Error("Cannot combine an upgrade with a streaming success response");
      }
      validateStreamSuccess(inner, method);
      if (hasStream) {
        throw new Error("Multiple streaming success responses are not supported");
      }
      hasStream = true;
      const entry = getStatusEntry(statuses, status);
      if (entry.noContent) {
        throw new Error(`Cannot combine no-content and streaming success responses for status: ${status}`);
      }
      for (const declared of inner.contentTypes) {
        if (entry.bufferedContentTypes.has(MediaType.normalize(declared))) {
          throw new Error(`Cannot combine buffered and streaming success responses for status ${status} and content-type: ${declared}`);
        }
      }
      statuses.set(status, {
        ...entry,
        stream: inner
      });
    } else {
      const entry = getStatusEntry(statuses, status);
      const noContent = HttpApiSchema.isNoContent(inner.ast);
      if (entry.stream !== undefined) {
        if (noContent) {
          throw new Error(`Cannot combine no-content and streaming success responses for status: ${status}`);
        }
        const encoding = HttpApiSchema.getResponseEncodingSchema(schema);
        const contentType = MediaType.normalize(encoding.contentType);
        if (entry.stream.contentTypes.some(declared => MediaType.normalize(declared) === contentType)) {
          throw new Error(`Cannot combine buffered and streaming success responses for status ${status} and content-type: ${encoding.contentType}`);
        }
      }
      if (!noContent) {
        entry.bufferedContentTypes.add(MediaType.normalize(HttpApiSchema.getResponseEncodingSchema(schema).contentType));
      }
      entry.noContent = entry.noContent || noContent;
    }
  }
  validateResponseExclusivity(schemas, HttpApiSchema.getStatusSuccessSchema);
}
function getStatusEntry(statuses, status) {
  let entry = statuses.get(status);
  if (entry === undefined) {
    entry = {
      bufferedContentTypes: new Set(),
      noContent: false
    };
    statuses.set(status, entry);
  }
  return entry;
}
function validateResponseExclusivity(schemas, getStatus) {
  const statuses = new Map();
  for (const schema of schemas) {
    const status = getStatus(schema);
    const withHeadersAnnotation = HttpApiSchema.getWithHeadersAnnotation(schema.ast);
    const body = HttpApiSchema.isWithHeaders(schema) ? schema.schema : withHeadersAnnotation?.body ?? schema;
    const contentType = HttpApiSchema.isNoContent(body.ast) ? "" : MediaType.normalize(HttpApiSchema.isStreamSchema(body) ? body.contentType : HttpApiSchema.getResponseEncodingSchema(schema).contentType);
    let entry = statuses.get(status);
    if (entry === undefined) {
      entry = {
        headerContentType: undefined,
        plainContentTypes: new Set()
      };
      statuses.set(status, entry);
    }
    const combineError = () => new Error(`Cannot combine a response with headers with another response for status ${status} and content-type: ${contentType || "<no content>"}`);
    if (HttpApiSchema.isWithHeaders(schema) || withHeadersAnnotation !== undefined) {
      if (entry.headerContentType !== undefined) {
        throw new Error(`Cannot declare multiple responses with headers for status ${status}`);
      }
      if (entry.plainContentTypes.has(contentType)) throw combineError();
      entry.headerContentType = contentType;
    } else {
      if (entry.headerContentType === contentType) throw combineError();
      entry.plainContentTypes.add(contentType);
    }
  }
}
function validateStreamSuccess(schema, method) {
  if (method === "HEAD") {
    throw new Error("HEAD endpoints cannot declare streaming success responses");
  }
  if (HttpApiSchema.isStreamSse(schema) && hasReservedSseEventName(schema.events.ast)) {
    throw new Error(`SSE event name is reserved: ${reservedStreamFailureEvent}`);
  }
}
function hasReservedSseEventName(ast) {
  return hasReservedEventName(AST.toEncoded(ast), new Set());
}
function hasReservedEventName(ast, seen) {
  if (seen.has(ast)) return false;
  seen.add(ast);
  if (AST.isUnion(ast)) {
    return ast.types.some(type => hasReservedEventName(type, seen));
  }
  if (AST.isSuspend(ast)) {
    return hasReservedEventName(ast.thunk(), seen);
  }
  if (!AST.isObjects(ast)) return false;
  const event = ast.propertySignatures.find(ps => ps.name === "event");
  return event !== undefined && hasReservedEventLiteral(event.type, seen);
}
function hasReservedEventLiteral(ast, seen) {
  if (seen.has(ast)) return false;
  seen.add(ast);
  const encoded = AST.toEncoded(ast);
  if (encoded !== ast) {
    return hasReservedEventLiteral(encoded, seen);
  }
  if (AST.isLiteral(ast)) {
    return ast.literal === reservedStreamFailureEvent;
  }
  if (AST.isUnion(ast)) {
    return ast.types.some(type => hasReservedEventLiteral(type, seen));
  }
  if (AST.isSuspend(ast)) {
    return hasReservedEventLiteral(ast.thunk(), seen);
  }
  return false;
}
function transformResponse(schema) {
  const encoding = HttpApiSchema.getResponseEncoding(schema.ast);
  const withHeaders = HttpApiSchema.getWithHeadersAnnotation(schema.ast);
  if (withHeaders === undefined) {
    return applyResponseEncoding(schema, encoding);
  }
  return schema.annotate({
    "~httpApiWithHeaders": {
      ...withHeaders,
      bodyCodec: applyResponseEncoding(withHeaders.body, encoding),
      headersCodec: toCodecHttpProperties(withHeaders.headers)
    }
  });
}
function applyResponseEncoding(schema, encoding) {
  switch (encoding._tag) {
    case "Json":
      return Schema.toCodecJson(schema);
    case "FormUrlEncoded":
      return Schema.toCodecStringTree(schema);
    case "Text":
    case "Uint8Array":
      return schema;
  }
}
function transformPayload(schema, method) {
  if (!hasBody(method)) return toCodecHttpProperties(schema);
  const encoding = HttpApiSchema.getPayloadEncoding(schema.ast, method);
  switch (encoding._tag) {
    case "Json":
      return Schema.toCodecJson(schema);
    case "FormUrlEncoded":
      return Schema.toCodecStringTree(schema);
    case "Text":
    case "Uint8Array":
    case "Multipart":
      return schema;
  }
}
/**
 * Creates a `GET` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const get = /*#__PURE__*/make("GET");
/**
 * Creates a `QUERY` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const query = /*#__PURE__*/make("QUERY");
/**
 * Creates a `POST` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const post = /*#__PURE__*/make("POST");
/**
 * Creates a `PUT` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const put = /*#__PURE__*/make("PUT");
/**
 * Creates a `PATCH` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const patch = /*#__PURE__*/make("PATCH");
const del = /*#__PURE__*/make("DELETE");
/**
 * @stability unstable
 */
export {
/**
 * Creates a `DELETE` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
del as delete };
/**
 * Creates a `HEAD` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const head = /*#__PURE__*/make("HEAD");
/**
 * Creates an `OPTIONS` endpoint declaration.
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.0
 */
export const options = /*#__PURE__*/make("OPTIONS");
//# sourceMappingURL=HttpApiEndpoint.js.map