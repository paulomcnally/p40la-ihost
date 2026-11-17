import * as Data from "../Data.js";
import * as Effect from "../Effect.js";
import * as ErrorReporter from "../ErrorReporter.js";
import * as HttpServerRespondable from "../http/HttpServerRespondable.js";
import * as HttpServerResponse from "../http/HttpServerResponse.js";
import * as ProblemDetails from "../http/ProblemDetails.js";
import { hasProperty } from "../Predicate.js";
import * as Schema from "../Schema.js";
import * as HttpApiSchema from "./HttpApiSchema.js";
const badRequestResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 400
});
const unauthorizedResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 401
});
const forbiddenResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 403
});
const notFoundResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 404
});
const methodNotAllowedResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 405
});
const notAcceptableResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 406
});
const requestTimeoutResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 408
});
const conflictResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 409
});
const goneResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 410
});
const payloadTooLargeResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 413
});
const unprocessableEntityResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 422
});
const internalServerErrorResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 500
});
const notImplementedResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 501
});
const serviceUnavailableResponse = /*#__PURE__*/HttpServerResponse.empty({
  status: 503
});
/**
 * Built-in HTTP API error for a `400 Bad Request` response. When used directly as
 * a server response, it renders as an empty response with status 400.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class BadRequest extends /*#__PURE__*/Schema.Error("effect/HttpApiError/BadRequest")({
  _tag: /*#__PURE__*/Schema.tag("BadRequest")
}, {
  description: "BadRequest",
  httpApiStatus: 400
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(badRequestResponse);
  }
  static singleton = /*#__PURE__*/new BadRequest();
}
/**
 * No-content schema variant for `BadRequest`, decoding an empty 400 response into
 * a `BadRequest` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const BadRequestNoContent = /*#__PURE__*/BadRequest.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new BadRequest({})
}));
/**
 * Built-in HTTP API error for a `401 Unauthorized` response. When used directly as
 * a server response, it renders as an empty response with status 401.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class Unauthorized extends /*#__PURE__*/Schema.Error("effect/HttpApiError/Unauthorized")({
  _tag: /*#__PURE__*/Schema.tag("Unauthorized")
}, {
  description: "Unauthorized",
  httpApiStatus: 401
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(unauthorizedResponse);
  }
}
/**
 * No-content schema variant for `Unauthorized`, decoding an empty 401 response
 * into an `Unauthorized` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const UnauthorizedNoContent = /*#__PURE__*/Unauthorized.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new Unauthorized({})
}));
/**
 * Built-in HTTP API error for a `403 Forbidden` response. When used directly as a
 * server response, it renders as an empty response with status 403.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class Forbidden extends /*#__PURE__*/Schema.Error("effect/HttpApiError/Forbidden")({
  _tag: /*#__PURE__*/Schema.tag("Forbidden")
}, {
  description: "Forbidden",
  httpApiStatus: 403
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(forbiddenResponse);
  }
}
/**
 * No-content schema variant for `Forbidden`, decoding an empty 403 response into a
 * `Forbidden` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const ForbiddenNoContent = /*#__PURE__*/Forbidden.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new Forbidden({})
}));
/**
 * Built-in HTTP API error for a `404 Not Found` response. When used directly as a
 * server response, it renders as an empty response with status 404.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class NotFound extends /*#__PURE__*/Schema.Error("effect/HttpApiError/NotFound")({
  _tag: /*#__PURE__*/Schema.tag("NotFound")
}, {
  description: "NotFound",
  httpApiStatus: 404
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(notFoundResponse);
  }
}
/**
 * No-content schema variant for `NotFound`, decoding an empty 404 response into a
 * `NotFound` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const NotFoundNoContent = /*#__PURE__*/NotFound.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new NotFound({})
}));
/**
 * Built-in HTTP API error for a `405 Method Not Allowed` response. When used
 * directly as a server response, it renders as an empty response with status 405.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class MethodNotAllowed extends /*#__PURE__*/Schema.Error("effect/HttpApiError/MethodNotAllowed")({
  _tag: /*#__PURE__*/Schema.tag("MethodNotAllowed")
}, {
  description: "MethodNotAllowed",
  httpApiStatus: 405
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(methodNotAllowedResponse);
  }
}
/**
 * No-content schema variant for `MethodNotAllowed`, decoding an empty 405 response
 * into a `MethodNotAllowed` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const MethodNotAllowedNoContent = /*#__PURE__*/MethodNotAllowed.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new MethodNotAllowed({})
}));
/**
 * Built-in HTTP API error for a `406 Not Acceptable` response. When used directly
 * as a server response, it renders as an empty response with status 406.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class NotAcceptable extends /*#__PURE__*/Schema.Error("effect/HttpApiError/NotAcceptable")({
  _tag: /*#__PURE__*/Schema.tag("NotAcceptable")
}, {
  description: "NotAcceptable",
  httpApiStatus: 406
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(notAcceptableResponse);
  }
}
/**
 * No-content schema variant for `NotAcceptable`, decoding an empty 406 response
 * into a `NotAcceptable` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const NotAcceptableNoContent = /*#__PURE__*/NotAcceptable.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new NotAcceptable({})
}));
/**
 * Built-in HTTP API error for a `408 Request Timeout` response. When used directly
 * as a server response, it renders as an empty response with status 408.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class RequestTimeout extends /*#__PURE__*/Schema.Error("effect/HttpApiError/RequestTimeout")({
  _tag: /*#__PURE__*/Schema.tag("RequestTimeout")
}, {
  description: "RequestTimeout",
  httpApiStatus: 408
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(requestTimeoutResponse);
  }
}
/**
 * No-content schema variant for `RequestTimeout`, decoding an empty 408 response
 * into a `RequestTimeout` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const RequestTimeoutNoContent = /*#__PURE__*/RequestTimeout.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new RequestTimeout({})
}));
/**
 * Built-in HTTP API error for a `409 Conflict` response. When used directly as a
 * server response, it renders as an empty response with status 409.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class Conflict extends /*#__PURE__*/Schema.Error("effect/HttpApiError/Conflict")({
  _tag: /*#__PURE__*/Schema.tag("Conflict")
}, {
  description: "Conflict",
  httpApiStatus: 409
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(conflictResponse);
  }
}
/**
 * No-content schema variant for `Conflict`, decoding an empty 409 response into a
 * `Conflict` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const ConflictNoContent = /*#__PURE__*/Conflict.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new Conflict({})
}));
/**
 * Built-in HTTP API error for a `410 Gone` response. When used directly as a
 * server response, it renders as an empty response with status 410.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class Gone extends /*#__PURE__*/Schema.Error("effect/HttpApiError/Gone")({
  _tag: /*#__PURE__*/Schema.tag("Gone")
}, {
  description: "Gone",
  httpApiStatus: 410
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(goneResponse);
  }
}
/**
 * No-content schema variant for `Gone`, decoding an empty 410 response into a
 * `Gone` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const GoneNoContent = /*#__PURE__*/Gone.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new Gone({})
}));
/**
 * Built-in HTTP API error for a `422 Unprocessable Entity` response. When used
 * directly as a server response, it renders as an empty response with status 422.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class UnprocessableEntity extends /*#__PURE__*/Schema.Error("effect/HttpApiError/UnprocessableEntity")({
  _tag: /*#__PURE__*/Schema.tag("UnprocessableEntity")
}, {
  description: "UnprocessableEntity",
  httpApiStatus: 422
}) {
  [ErrorReporter.ignore] = true;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(unprocessableEntityResponse);
  }
}
/**
 * No-content schema variant for `UnprocessableEntity`, decoding an empty 422
 * response into an `UnprocessableEntity` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const UnprocessableEntityNoContent = /*#__PURE__*/UnprocessableEntity.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new UnprocessableEntity({})
}));
/**
 * Built-in HTTP API error for a `500 Internal Server Error` response. When used
 * directly as a server response, it renders as an empty response with status 500.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class InternalServerError extends /*#__PURE__*/Schema.Error("effect/HttpApiError/InternalServerError")({
  _tag: /*#__PURE__*/Schema.tag("InternalServerError")
}, {
  description: "InternalServerError",
  httpApiStatus: 500
}) {
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(internalServerErrorResponse);
  }
}
/**
 * No-content schema variant for `InternalServerError`, decoding an empty 500
 * response into an `InternalServerError` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const InternalServerErrorNoContent = /*#__PURE__*/InternalServerError.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new InternalServerError({})
}));
/**
 * Built-in HTTP API error for a `501 Not Implemented` response. When used directly
 * as a server response, it renders as an empty response with status 501.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class NotImplemented extends /*#__PURE__*/Schema.Error("effect/HttpApiError/NotImplemented")({
  _tag: /*#__PURE__*/Schema.tag("NotImplemented")
}, {
  description: "NotImplemented",
  httpApiStatus: 501
}) {
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(notImplementedResponse);
  }
}
/**
 * No-content schema variant for `NotImplemented`, decoding an empty 501 response
 * into a `NotImplemented` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const NotImplementedNoContent = /*#__PURE__*/NotImplemented.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new NotImplemented({})
}));
/**
 * Built-in HTTP API error for a `503 Service Unavailable` response. When used
 * directly as a server response, it renders as an empty response with status 503.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class ServiceUnavailable extends /*#__PURE__*/Schema.Error("effect/HttpApiError/ServiceUnavailable")({
  _tag: /*#__PURE__*/Schema.tag("ServiceUnavailable")
}, {
  description: "ServiceUnavailable",
  httpApiStatus: 503
}) {
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(serviceUnavailableResponse);
  }
}
/**
 * No-content schema variant for `ServiceUnavailable`, decoding an empty 503
 * response into a `ServiceUnavailable` error value.
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.0
 */
export const ServiceUnavailableNoContent = /*#__PURE__*/ServiceUnavailable.pipe(/*#__PURE__*/HttpApiSchema.asNoContent({
  decode: () => new ServiceUnavailable({})
}));
const problemEncoding = {
  _tag: "Json",
  contentType: ProblemDetails.contentType
};
/**
 * Declares a typed RFC 9457 problem as a yieldable error class.
 *
 * **Details**
 *
 * The class is a `Schema.Error` whose `_tag` is the given tag, so `Effect.catchTag`
 * works without `_tag` on the wire. It carries the `httpApiStatus` and
 * `application/problem+json` annotations, so endpoints that declare it as an
 * error render, document and decode it as a problem. Problem classes that
 * share a status are told apart by `type`. The extension fields are validated
 * on both sides and cannot replace `type`, `status` or `title`; unknown
 * extensions are not kept. An undeclared instance still renders as a problem
 * through `HttpServerRespondable`.
 *
 * **Example** (Declaring and using a typed problem)
 *
 * ```ts
 * import { Effect, Schema } from "effect"
 * import { HttpApiEndpoint, HttpApiError } from "effect/http-api"
 *
 * class OutOfCredit extends HttpApiError.Problem<OutOfCredit>()("OutOfCredit", {
 *   type: "https://example.com/probs/out-of-credit",
 *   status: 403,
 *   title: "You do not have enough credit."
 * }, {
 *   balance: Schema.Number
 * }) {}
 *
 * const buy = HttpApiEndpoint.post("buy", "/buy", { error: OutOfCredit })
 *
 * const failing = Effect.fail(new OutOfCredit({ balance: 30 }))
 * // Responds with 403 and { "type": "https://example.com/probs/out-of-credit",
 * //   "status": 403, "title": "You do not have enough credit.", "balance": 30 }
 * ```
 *
 * @stability unstable
 * @category constructors
 * @since 4.0.6
 */
export const Problem = () => (tag, declaration, fields, annotations) => {
  const struct = Schema.Struct({
    detail: ProblemDetails.optionalMember(Schema.String),
    instance: ProblemDetails.optionalMember(Schema.String),
    ...fields,
    type: Schema.tag(declaration.type),
    status: ProblemDetails.fixedMember(declaration.status),
    title: ProblemDetails.fixedMember(declaration.title)
  });
  const encode = Schema.encodeUnknownEffect(struct);
  const Base = Schema.Error(tag)(struct, {
    description: declaration.title,
    httpApiStatus: declaration.status,
    "~httpApiEncoding": problemEncoding,
    ...annotations
  });
  Object.assign(Base.prototype, {
    _tag: tag,
    [ErrorReporter.ignore]: declaration.status < 500,
    [HttpServerRespondable.symbol]() {
      return Effect.map(encode(this), body => HttpServerResponse.jsonUnsafe(body, {
        status: declaration.status,
        contentType: ProblemDetails.contentType
      }));
    }
  });
  Object.defineProperty(Base.prototype, "message", {
    get() {
      return this.detail ?? declaration.title;
    },
    configurable: true
  });
  return Base;
};
/**
 * One entry of `BadRequestProblem.errors`: the location of a request value
 * that failed decoding and the reason.
 *
 * **Details**
 *
 * The first path segment names the request slot (`params`, `headers`, `query`
 * or `payload`).
 *
 * @stability unstable
 * @category schemas
 * @since 4.0.6
 */
export const ProblemIssue = /*#__PURE__*/Schema.Struct({
  path: /*#__PURE__*/Schema.Array(/*#__PURE__*/Schema.Union([Schema.String, Schema.Number])),
  message: Schema.String
});
/**
 * The problem rendered by `HttpApiMiddleware.layerProblemDetails` when a
 * request fails decoding.
 *
 * **Details**
 *
 * The body is generic by default. `errors` is only filled when the layer is
 * built with `includeDetails: true`.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.6
 */
export class BadRequestProblem extends /*#__PURE__*/Problem()("BadRequestProblem", {
  type: ProblemDetails.blankType,
  status: 400,
  title: "Bad Request"
}, {
  errors: /*#__PURE__*/Schema.optionalKey(/*#__PURE__*/Schema.Array(ProblemIssue))
}) {}
/**
 * The problem rendered by `HttpApiMiddleware.layerProblemDetails` when a
 * response fails encoding.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.6
 */
export class InternalServerErrorProblem extends /*#__PURE__*/Problem()("InternalServerErrorProblem", {
  type: ProblemDetails.blankType,
  status: 500,
  title: "Internal Server Error"
}) {}
/**
 * Runtime identifier used to mark and detect `HttpApiSchemaError` values.
 *
 * @stability unstable
 * @category type IDs
 * @since 4.0.0
 */
export const HttpApiSchemaErrorTypeId = "~effect/http-api/HttpApiError/HttpApiSchemaError";
/**
 * Error raised when an HTTP API request or response component fails schema
 * decoding or encoding. Request decoding failures respond with an empty
 * `400 Bad Request` and are ignored by error reporters. Response encoding
 * failures respond with an empty `500 Internal Server Error` and are reported.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.0
 */
export class HttpApiSchemaError extends /*#__PURE__*/Data.TaggedClass("HttpApiSchemaError") {
  [HttpApiSchemaErrorTypeId] = HttpApiSchemaErrorTypeId;
  get [ErrorReporter.ignore]() {
    return this.kind !== "Body" && this.kind !== "ResponseHeaders";
  }
  static is(u) {
    return hasProperty(u, HttpApiSchemaErrorTypeId);
  }
  static wrap(kind, effect) {
    // Decoders return completed Exit values for synchronous results, so map
    // eagerly and skip the error wrapper when nothing remains to run.
    return Effect.mapErrorEager(effect, error => new HttpApiSchemaError({
      kind,
      cause: error
    }));
  }
  name = "HttpApiSchemaError";
  message = this.kind;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(this[ErrorReporter.ignore] ? badRequestResponse : internalServerErrorResponse);
  }
}
/**
 * Runtime identifier used to mark and detect `HttpApiPayloadTooLarge` values.
 *
 * @stability unstable
 * @category type IDs
 * @since 4.0.6
 */
export const HttpApiPayloadTooLargeTypeId = "~effect/http-api/HttpApiError/HttpApiPayloadTooLarge";
/**
 * Error raised when a streaming request payload exceeds a limit declared on
 * its `HttpApiSchema.StreamUint8Array` or `HttpApiSchema.StreamNdjson` schema.
 * It responds with an empty `413 Payload Too Large` and is ignored by error
 * reporters.
 *
 * **Details**
 *
 * `limit` names the option that was exceeded: `maxBytes` caps the whole body
 * and `maxElementBytes` caps one NDJSON line. `size` is its configured value.
 *
 * @stability unstable
 * @category errors
 * @since 4.0.6
 */
export class HttpApiPayloadTooLarge extends /*#__PURE__*/Data.TaggedClass("HttpApiPayloadTooLarge") {
  [HttpApiPayloadTooLargeTypeId] = HttpApiPayloadTooLargeTypeId;
  [ErrorReporter.ignore] = true;
  static is(u) {
    return hasProperty(u, HttpApiPayloadTooLargeTypeId);
  }
  name = "HttpApiPayloadTooLarge";
  message = `Payload exceeds ${this.limit} of ${this.size} bytes`;
  [HttpServerRespondable.symbol]() {
    return Effect.succeed(payloadTooLargeResponse);
  }
}
//# sourceMappingURL=HttpApiError.js.map